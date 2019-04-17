package ping

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRunPingerNoAddress(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(time.Second)
		cancel()
	}()
	p, err := NewPinger(ctx)
	handleNoErr(t, err)
	p.Run()
}

func TestRunPingerWrongAddress(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(time.Second)
		cancel()
	}()
	p, err := NewPinger(ctx, "some.wrong.address", "localhost")
	handleErr(t, err)
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

func handleNoErr(t *testing.T, err error) {
	assert.NoError(t, err)
	if err != nil {
		t.FailNow()
	}
}

func handleErr(t *testing.T, err error) {
	assert.Error(t, err)
	if err == nil {
		t.FailNow()
	}
}
