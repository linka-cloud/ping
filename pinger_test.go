package ping

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestRunPingerNoAddress(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(time.Second)
		cancel()
	}()
	p, err := NewPinger(ctx)
	assertNoErr(t, err)
	p.Run()
}

func TestRunPingerWrongAddress(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(time.Second)
		cancel()
	}()
	p, err := NewPinger(ctx, "some.wrong.address", "localhost")
	assertErr(t, err)
	assert.Nil(t, p)
}

func TestPingerContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(500 * time.Millisecond)
		cancel()
	}()
	p, err := newPinger(ctx)
	assertNoErr(t, err)
	go p.Run()
	time.Sleep(time.Second)
	assert.False(t, p.IsRunning())
}

func TestPingerLocalhost(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(5 * time.Second)
		cancel()
	}()
	ip := "127.0.0.1"
	p, err := NewPinger(ctx)
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
			s := p.Statistics()
			assert.NotEmpty(t, s)
			l, ok := s[ip]
			assert.True(t, ok)
			logrus.Debug(l)
			assert.Equal(t, count, l.PacketsRecv)
			rtts := filterZeros(l.Rtts)
			assert.Equal(t, count, len(rtts))
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
	p, err := NewPinger(ctx)
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
			s := p.Statistics()
			assert.NotEmpty(t, s)
			l, ok := s[ip]
			assert.True(t, ok)
			logrus.Debug(l)
			assert.Equal(t, 0, l.PacketsRecv)
			rtts := filterZeros(l.Rtts)
			assert.Equal(t, 0, len(rtts))
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
	p, err := NewPinger(ctx)
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
			s := p.Statistics()
			assert.NotEmpty(t, s)

			l, ok := s[ip1]
			assert.True(t, ok)
			logrus.Debug(l)
			assert.Equal(t, 0, l.PacketsRecv)
			rtts := filterZeros(l.Rtts)
			assert.Equal(t, 0, len(rtts))

			l, ok = s[ip2]
			assert.True(t, ok)
			logrus.Debug(l)
			assert.InDelta(t, count, l.PacketsRecv, 1)
			rtts = filterZeros(l.Rtts)
			assert.InDelta(t, l.PacketsRecv, len(rtts), 1)
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

func filterZeros(s []time.Duration) []time.Duration {
	var out []time.Duration
	for _, d := range s {
		if d != time.Duration(0) {
			out = append(out, d)
		}
	}
	return out
}

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

func TestPinger(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	p, err := NewPinger(ctx, "127.0.0.1", "255.0.0.255", "127.0.0.2")
	assert.NoError(t, err)
	if err != nil {
		t.FailNow()
	}
	go func() {
		time.Sleep(10 * time.Second)
		cancel()
	}()

	go func() {
		time.Sleep(5 * time.Second)
		logrus.Debug("removing 127.0.0.2")
		err := p.RemoveAddress("127.0.0.2")
		assert.NoError(t, err)
		logrus.Debug("127.0.0.2 removed")
		as := p.Addresses()
		assert.NotContains(t, as, "127.0.0.2")
	}()
	assert.True(t, p.Privileged())
	as := p.Addresses()
	assert.Equal(t, 3, len(as))
	assert.Contains(t, as, "127.0.0.1")
	assert.Contains(t, as, "127.0.0.2")
	assert.Contains(t, as, "255.0.0.255")
	err = p.AddAddress("127.0.0.1")
	assert.Error(t, err)
	tk := time.NewTicker(time.Second)
	defer tk.Stop()
	count := 0
	go p.Run()
	for {
		select {
		case <-tk.C:
			count++
			s := p.Statistics()
			logrus.Debugf("count: %d", count)
			for _, v := range s {
				logStats(v)
				for _, d := range v.Rtts {
					assert.True(t, d.Seconds() < 1)
				}
			}
			if count < 5 {
				assert.Equal(t, 3, len(s))

				assert.InDelta(t, count, s["127.0.0.1"].PacketsSent, 1)
				assert.InDelta(t, count, s["127.0.0.1"].PacketsRecv, 1)
				assert.Equal(t, float64(0), s["127.0.0.1"].PacketLoss)

				assert.InDelta(t, count, s["127.0.0.2"].PacketsSent, 1)
				assert.InDelta(t, count, s["127.0.0.2"].PacketsRecv, 1)
				assert.Equal(t, float64(0), s["127.0.0.2"].PacketLoss)

				assert.InDelta(t, count, s["255.0.0.255"].PacketsSent, 1)
				assert.Equal(t, 0, s["255.0.0.255"].PacketsRecv)
				assert.Equal(t, float64(100), s["255.0.0.255"].PacketLoss)
			} else if count > 6 {
				assert.Equal(t, 2, len(s))

				assert.InDelta(t, count, s["127.0.0.1"].PacketsSent, 1)

				assert.InDelta(t, count, s["255.0.0.255"].PacketsSent, 1)
				assert.Equal(t, 0, s["255.0.0.255"].PacketsRecv)
				assert.Equal(t, float64(100), s["255.0.0.255"].PacketLoss)

				assert.NotContains(t, s, "127.0.0.2")
			}

		case <-ctx.Done():
			return
		}
	}
}

func TestPingerStatistics(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	p, err := NewPinger(ctx, "127.0.0.1")
	assert.NoError(t, err)
	if err != nil {
		t.FailNow()
	}
	go func() {
		time.Sleep(5 * time.Second)
		cancel()
	}()

	tk := time.NewTicker(time.Second)
	defer tk.Stop()
	count := 0
	go p.Run()
	for {
		select {
		case <-tk.C:
			count++
			s := p.Statistics()
			ls, ok := s["127.0.0.1"]
			assert.True(t, ok)
			logrus.Debugf("127.0.0.1 : %v", ls)
			assert.Equal(t, 10, len(ls.Rtts))
			for i, rtt := range ls.Rtts {
				if i > 10-count {
					assert.Equal(t, time.Duration(0), rtt)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

//func TestMaxint(t *testing.T) {
//	c := math.MaxInt64
//	var i int64 = 0
//	for ; i < 10; i++ {
//		c++
//		logrus.Debug(c)
//	}
//}
