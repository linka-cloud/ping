package ping

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/digineo/go-ping"
	"github.com/sirupsen/logrus"
)

//Pinger
type Pinger interface {
	// deprecated: removed udp support
	Privileged() bool

	// Addresses returns the list of the destinations host
	Addresses() []string
	// IPAddresses returns the list of the destinations addresses
	IPAddresses() []net.IPAddr

	// AddAddress add a destination to the pinger
	AddAddress(a string) error
	// Remove Address remove a destination from the pinger
	RemoveAddress(a string) error

	// Run start the pinger. It fails silently if the pinger is already running
	Run()
	// Stop stops the pinger. It fails silently if the pinger is already stopped
	Stop()

	// IsRunning returns the state of the pinger
	IsRunning() bool

	// Statistics returns the a map address ping Statistics
	Statistics() map[string]Statistics

	// Close closes the connection. It should be call deferred right after the creation of the pinger
	Close()
}

type _pinger struct {
	ctx context.Context

	opts   options
	pinger *ping.Pinger

	dsts map[string]*destination
	dmu  sync.RWMutex

	running bool
	rmu     sync.RWMutex

	clients []chan map[string]Statistics
	stats   map[string]Statistics
	smu     sync.RWMutex

	done chan bool
}

//NewPinger create a new Pinger with given addresses
func NewPinger(ctx context.Context, addrs ...string) (Pinger, error) {
	p, err := newPinger(ctx)
	if err != nil {
		return nil, err
	}
	for _, a := range addrs {
		if err := p.AddAddress(a); err != nil {
			return nil, err
		}
	}
	return p, nil
}

func newPinger(ctx context.Context) (*_pinger, error) {
	p := &_pinger{
		ctx:   ctx,
		opts:  defaultOptions,
		dsts:  make(map[string]*destination),
		stats: make(map[string]Statistics),
		done:  make(chan bool),
	}
	instance, err := ping.New(defaultOptions.bind4, defaultOptions.bind6)
	if err != nil {
		return nil, err
	}
	if instance.PayloadSize() != uint16(defaultOptions.payloadSize) {
		instance.SetPayloadSize(uint16(defaultOptions.payloadSize))
	}
	p.pinger = instance
	return p, nil
}

// deprecated
func (*_pinger) Privileged() bool {
	return true
}

func (p *_pinger) Addresses() []string {
	p.dmu.Lock()
	var as []string
	for k := range p.dsts {
		as = append(as, k)
	}
	p.dmu.Unlock()
	return as
}

func (p *_pinger) IPAddresses() []net.IPAddr {
	p.dmu.RLock()
	var ipas []net.IPAddr
	for _, v := range p.dsts {
		ipas = append(ipas, *v.remote)
	}
	p.dmu.RUnlock()
	return ipas
}

func (p *_pinger) AddAddress(a string) error {
	p.dmu.Lock()
	defer p.dmu.Unlock()
	ipaddr, err := resolve(a, p.opts.resolverTimeout)
	if err != nil {
		return fmt.Errorf("error resolving host %s: %v", a, err)
	}
	if _, ok := p.dsts[a]; ok {
		return fmt.Errorf("%s already exists", a)
	}
	if v4 := ipaddr.IP.To4() != nil; v4 && p.opts.bind4 == "" || !v4 && p.opts.bind6 == "" {
		return errors.New("configuration cannot support this ip, please set bind4 or bind6")
	}

	dst := destination{
		host:   a,
		remote: &ipaddr,
		history: &history{
			results: make([]time.Duration, p.opts.statBufferSize),
		},
	}

	p.dsts[a] = &dst
	return nil
}

func (p *_pinger) RemoveAddress(a string) error {
	p.dmu.Lock()
	defer p.dmu.Unlock()
	if _, ok := p.dsts[a]; !ok {
		return fmt.Errorf("%s not found", a)
	}
	delete(p.dsts, a)
	p.smu.Lock()
	delete(p.stats, a)
	p.smu.Unlock()
	return nil
}

func (p *_pinger) Run() {
	p.rmu.Lock()
	p.running = true
	p.rmu.Unlock()
	p.ping()
	t := time.NewTicker(p.opts.interval)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			p.ping()
		case <-p.ctx.Done():
			logrus.Debug(p.ctx.Err())
			p.Stop()
		case <-p.done:
			logrus.Debug("received stop signal")
			return
		}
	}
}

func (p *_pinger) ping() {
	p.dmu.RLock()
	logrus.Debugf("destinations' count: %d", len(p.dsts))
	for a := range p.dsts {
		go func(d *destination, addr string) {
			logrus.Debugf("pinging %s", addr)
			d.ping(p.pinger, p.opts.timeout)
			s := d.compute()
			s.Addr = addr
			s.IPAddr = *d.remote
			p.smu.Lock()
			p.stats[addr] = s
			p.smu.Unlock()
		}(p.dsts[a], a)
	}
	p.dmu.RUnlock()
}

func (p *_pinger) Stop() {
	p.rmu.Lock()
	defer p.rmu.Unlock()
	if !p.running {
		return
	}
	close(p.done)
	p.running = false
}

func (p *_pinger) IsRunning() bool {
	p.rmu.RLock()
	defer p.rmu.RUnlock()
	return p.running
}

func (p *_pinger) Statistics() map[string]Statistics {
	p.smu.RLock()
	p.dmu.Lock()
	// Empty Copy
	out := make(map[string]Statistics)
	// Filter removed addresses
	for k := range p.stats {
		if _, ok := p.dsts[k]; !ok {
			delete(p.stats, k)
		} else {
			// Copy to output map
			s := p.stats[k]
			s.Rtts = make([]time.Duration, 10)
			copy(s.Rtts, p.stats[k].Rtts)
			out[k] = s
		}
	}
	p.dmu.Unlock()
	defer p.smu.RUnlock()
	return out
}

func (p *_pinger) Close() {
	if p.IsRunning() {
		p.Stop()
	}
	p.pinger.Close()
}
