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

type Pinger interface {
	AddAddress(a string) error
	RemoveAddress(a string) error
	Run()
	Stop()
	Status() map[string]gping.Statistics
}

type GoPinger struct {
	ctx        context.Context
	running    atomic.Value
	targets    sync.Map
	results    sync.Map
	privileged bool
}

type target struct {
	*gping.Pinger
	running atomic.Value
}

func (t *target) Run() {
	t.running.Store(true)
	t.Pinger.Run()
}

func (t *target) Stop() {
	if !(t.running.Load().(bool)) {
		return
	}
	// The pinger may not be running raising a "close on closed channel error"
	// We just ignore if
	defer func() {
		recover()
	}()
	t.running.Store(false)
	t.Pinger.Stop()
}

func NewPinger(ctx context.Context, privileged bool, address ...string) (Pinger, error) {
	p := &GoPinger{
		ctx:        ctx,
		running:    atomic.Value{},
		targets:    sync.Map{},
		results:    sync.Map{},
		privileged: privileged,
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
	t := &target{pg, atomic.Value{}}
	t.running.Store(false)
	t.Timeout = time.Second
	t.Interval = time.Second
	t.Count = 100000
	t.SetPrivileged(p.privileged)
	t.OnRecv = func(packet *gping.Packet) {
		p.results.Store(a, *t.Statistics())
	}
	t.OnFinish = func(statistics *gping.Statistics) {
		p.results.Store(a, *t.Statistics())
		time.Sleep(time.Second)
		if t.running.Load().(bool) {
			t.Run()
		}
	}
	p.targets.Store(a, t)
	running := (p.running.Load()).(bool)
	if running {
		go t.Run()
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
	p.running.Store(false)
}

func (p *GoPinger) Status() map[string]gping.Statistics {
	rs := make(map[string]gping.Statistics)
	p.results.Range(func(key, value interface{}) bool {
		r, ok := value.(gping.Statistics)
		if !ok {
			return false
		}
		rs[key.(string)] = r
		return true
	})
	return rs
}
