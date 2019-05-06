package ping

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
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
		time.Sleep(5 * time.Second)
		cancel()
	}()
	ip := "127.0.0.1"
	p, err := NewPinger(ctx, privileged)
	assertNoErr(t, err)
	go p.Run()
	err = p.AddAddress(ip)
	assertNoErr(t, err)
	c := time.NewTicker(time.Second)
	count := 0
	for {
		select {
		case <-c.C:
			count++
			s := p.Status()
			assert.NotEmpty(t, s)
			l, ok := s[ip]
			assert.True(t, ok)
			fmt.Println(l)
			assert.Equal(t, count, l.PacketsRecv)
		case <-ctx.Done():
			return
		}
	}
}

func TestPingerTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(5 * time.Second)
		cancel()
	}()
	ip := "255.0.0.255"
	p, err := NewPinger(ctx, privileged)
	assertNoErr(t, err)
	go p.Run()
	err = p.AddAddress(ip)
	assertNoErr(t, err)
	c := time.NewTicker(time.Second)
	for {
		select {
		case <-c.C:
			s := p.Status()
			assert.NotEmpty(t, s)
			l, ok := s[ip]
			assert.True(t, ok)
			fmt.Println(l)
			assert.Equal(t, 0, l.PacketsRecv)
		case <-ctx.Done():
			return
		}
	}
}

func TestTwoIPs(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(5 * time.Second)
		cancel()
	}()
	ip1 := "255.0.0.255"
	ip2 := "127.0.0.1"
	p, err := NewPinger(ctx, privileged)
	assertNoErr(t, err)
	go p.Run()
	err = p.AddAddress(ip1)
	assertNoErr(t, err)
	err = p.AddAddress(ip2)
	assertNoErr(t, err)
	c := time.NewTicker(time.Second)
	count := 0
	for {
		select {
		case <-c.C:
			count++
			s := p.Status()
			assert.NotEmpty(t, s)
			l, ok := s[ip1]
			assert.True(t, ok)
			fmt.Println(l)
			assert.Equal(t, 0, l.PacketsRecv)
			l, ok = s[ip2]
			assert.True(t, ok)
			fmt.Println(l)
			assert.Equal(t, count, l.PacketsRecv)
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

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}
