package ping

import (
	"context"
	"errors"
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

	// Reset reset stop and remove all addresses
	Reset()

	// Statistics returns the a map address ping Statistics
	Statistics() map[string]Statistics

	SetLogger(l logrus.FieldLogger)

	// Close closes the connection. It should be call deferred right after the creation of the pinger
	Close()
}

var (
	ErrIPUnsupported = errors.New("configuration cannot support this ip, please set bind4 or bind6")
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
	ErrResolveHost   = errors.New("error resolving host")
)

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

	logger logrus.FieldLogger
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
	p.logger = logrus.New()
	return p, nil
}

// deprecated
func (*_pinger) Privileged() bool {
	return true
}

func (p *_pinger) SetLogger(l logrus.FieldLogger) {
	p.logger = l
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
		return ErrResolveHost
	}
	if _, ok := p.dsts[a]; ok {
		return ErrAlreadyExists
	}
	if v4 := ipaddr.IP.To4() != nil; v4 && p.opts.bind4 == "" || !v4 && p.opts.bind6 == "" {
		return ErrIPUnsupported
	}

	dst := destination{
		host:   a,
		remote: &ipaddr,
		history: &history{
			results: make([]time.Duration, p.opts.statBufferSize),
		},
	}
	p.logger.Debugf("Added address: %s", a)
	p.dsts[a] = &dst
	return nil
}

func (p *_pinger) RemoveAddress(a string) error {
	p.dmu.Lock()
	defer p.dmu.Unlock()
	if _, ok := p.dsts[a]; !ok {
		return ErrNotFound
	}
	delete(p.dsts, a)
	p.smu.Lock()
	delete(p.stats, a)
	p.logger.Debugf("Removed address %s (%v)", a, p.stats[a])
	p.smu.Unlock()
	return nil
}

func (p *_pinger) Run() {
	if p.IsRunning() {
		return
	}
	p.rmu.Lock()
	p.running = true
	p.done = make(chan bool)
	p.rmu.Unlock()
	p.logger.Debug("Starting pinger")
	p.ping()
	t := time.NewTicker(p.opts.interval)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			p.logger.Debug("Pinging")
			p.ping()
		case <-p.ctx.Done():
			p.logger.Debug(p.ctx.Err())
			p.Stop()
		case <-p.done:
			p.logger.Debug("received stop signal")
			return
		}
	}
}

func (p *_pinger) ping() {
	p.dmu.RLock()
	p.logger.Debugf("destinations' count: %d", len(p.dsts))
	p.logger.Debugf("destinations: %v", p.dsts)
	for a := range p.dsts {
		go func(d *destination, addr string) {
			p.logger.Debugf("pinging %s", addr)
			d.ping(p.pinger, p.opts.timeout)
			s := d.compute()
			p.logger.Debugf("%s : %v", addr, s)
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
	defer func() {
		recover()
	}()
	p.rmu.Lock()
	defer p.rmu.Unlock()
	if !p.running {
		return
	}
	close(p.done)
	p.running = false
	p.smu.Lock()
	p.stats = make(map[string]Statistics)
	p.smu.Unlock()
	p.dmu.Lock()
	for k := range p.dsts {
		p.dsts[k].history = &history{
			results: make([]time.Duration, p.opts.statBufferSize),
		}
	}
	p.dmu.Unlock()
	p.logger.Debug("Stopped")
}

func (p *_pinger) IsRunning() bool {
	p.rmu.RLock()
	defer p.rmu.RUnlock()
	return p.running
}

func (p *_pinger) Statistics() map[string]Statistics {
	p.logger.Debug("Collecting stats")
	p.smu.RLock()
	defer p.smu.RUnlock()
	p.dmu.Lock()
	defer p.dmu.Unlock()
	// Empty Copy
	out := make(map[string]Statistics)
	// Filter removed addresses
	for k := range p.stats {
		// Copy to output map
		if _, ok := p.dsts[k]; !ok {
			p.logger.Debugf("%s has been removed. deleting stats", k)
			delete(p.stats, k)
		} else {
			s := p.stats[k]
			p.logger.Debugf("Got stats for %s : rtts: %v", s.Addr, s.Rtts)
			s.Rtts = make([]time.Duration, 10)
			copy(s.Rtts, p.stats[k].Rtts)
			out[k] = s
		}
	}
	return out
}

func (p *_pinger) Reset() {
	p.Stop()
	p.dmu.Lock()
	defer p.dmu.Unlock()
	p.dsts = make(map[string]*destination)
	p.stats = make(map[string]Statistics)
	p.done = make(chan bool)
}

func (p *_pinger) Close() {
	if p.IsRunning() {
		p.Stop()
	}
	p.pinger.Close()
}
