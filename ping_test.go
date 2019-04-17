package ping

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	privileged = runtime.GOOS == "windows"
)

func TestRunPingerNoAddress(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(time.Second)
		cancel()
	}()
	p, err := NewPinger(ctx, privileged)
	assertNoErr(t, err)
	p.Run()
}

func TestRunPingerWrongAddress(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(time.Second)
		cancel()
	}()
	p, err := NewPinger(ctx, privileged, "some.wrong.address", "localhost")
	assertErr(t, err)
	assert.Nil(t, p)
}

func TestPingerContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(500 * time.Millisecond)
		cancel()
	}()
	p := GoPinger{ctx: ctx}
	go p.Run()
	time.Sleep(time.Second)
	assert.False(t, p.running.Load().(bool))
}

func TestPingerLocalhost(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(2 * time.Second)
		cancel()
	}()
	p, err := NewPinger(ctx, privileged, "localhost")
	assertNoErr(t, err)
	go p.Run()
	c := time.NewTicker(time.Second)
	for {
		select {
		case <-c.C:
			s := p.Status()
			assert.NotEmpty(t, s)
			l, ok := s["localhost"]
			assert.True(t, ok)
			assert.Equal(t, 1, l.PacketsRecv)
		case <-ctx.Done():
			return
		}
	}
}

func assertNoErr(t *testing.T, err error) {
	assert.NoError(t, err)
	if err != nil {
		t.FailNow()
	}
}

func assertErr(t *testing.T, err error) {
	assert.Error(t, err)
	if err == nil {
		t.FailNow()
	}
}
