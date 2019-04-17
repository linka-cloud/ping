package ping

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	gping "github.com/sparrc/go-ping"
)

type PingResult struct {
	address string
	ping    bool
}

type Pinger interface {
	AddAddress(a string) error
	RemoveAddress(a string) error
	Run()
	Stop()
	Status() []PingResult
}

type GoPinger struct {
	ctx     context.Context
	running atomic.Value
	targets sync.Map
	results sync.Map
}

type target struct {
	*gping.Pinger
	running bool
}

func (t *target) Run() {
	t.running = true
	t.Pinger.Run()
}

func (t *target) Stop() {
	if !t.running {
		return
	}
	// The pinger may not be running raising a "close on closed channel error"
	// We just ignore if
	defer func() {
		recover()
	}()
	t.running = false
	t.Pinger.Stop()
}

func NewPinger(ctx context.Context, address ...string) (Pinger, error) {
	p := &GoPinger{
		ctx:     ctx,
		running: atomic.Value{},
		targets: sync.Map{},
		results: sync.Map{},
	}
	p.running.Store(false)
	for _, a := range address {
		if err := p.AddAddress(a); err != nil {
			return nil, err
		}
	}
	go func() {
		<-ctx.Done()
		running := (p.running.Load()).(bool)
		if !running {
			return
		}

		p.targets.Range(func(key, value interface{}) bool {
			target, ok := value.(*target)
			if !ok {
				fmt.Println("Error: target in not a *target")
				return true
			}
			target.Stop()
			return true
		})
	}()
	return p, nil
}

func (p *GoPinger) AddAddress(a string) error {
	if _, ok := p.targets.Load(a); ok {
		return fmt.Errorf("%s already exists", a)
	}
	pg, err := gping.NewPinger(a)
	if err != nil {
		return err
	}
	t := &target{pg, false}
	t.Timeout = time.Second
	t.Count = 100000
	t.OnRecv = func(packet *gping.Packet) {
		p.results.Store(a, PingResult{
			address: a,
			ping:    true,
		})
	}
	t.OnFinish = func(statistics *gping.Statistics) {
		p.results.Store(a, PingResult{
			address: a,
			ping:    statistics.PacketsRecv != 0,
		})
		time.Sleep(time.Second)
		if t.running {
			t.Run()
		}
	}
	p.targets.Store(a, t)
	running := (p.running.Load()).(bool)
	if running {
		t.Run()
	}
	return nil
}

func (p *GoPinger) RemoveAddress(a string) error {
	mp, ok := p.targets.Load(a)
	if !ok {
		return fmt.Errorf("%s not found", a)
	}
	target, ok := mp.(*target)
	if !ok {
		return errors.New("WTF !!")
	}
	running := (p.running.Load()).(bool)
	if running {
		target.Stop()
	}
	p.targets.Delete(a)
	p.results.Delete(a)
	return nil
}

func (p *GoPinger) Run() {
	p.targets.Range(func(key, value interface{}) bool {
		target, ok := value.(*target)
		if !ok {
			fmt.Println("Error: target in not a *target")
			return true
		}
		go target.Run()
		return true
	})
	p.running.Store(true)
	<-p.ctx.Done()
	p.Stop()
}

func (p *GoPinger) Stop() {
	running := (p.running.Load()).(bool)
	if running {
		return
	}
	p.targets.Range(func(key, value interface{}) bool {
		target, ok := value.(*target)
		if !ok {
			fmt.Println("Error: target in not a *target")
			return true
		}
		target.Stop()
		return true
	})
	p.running.Store(false)
}

func (p *GoPinger) Status() []PingResult {
	var rs []PingResult
	p.results.Range(func(key, value interface{}) bool {
		r, ok := value.(PingResult)
		if !ok {
			return false
		}
		rs = append(rs, r)
		return true
	})
	return rs
}
