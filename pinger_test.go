package ping

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunPingerNoAddress(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	p, err := NewPinger(ctx)
	require.NoError(t, err)
	p.Run()
}

func TestRunPingerWrongAddress(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	p, err := NewPinger(ctx, "some.wrong.address", "localhost")
	assert.Error(t, err)
	assert.Nil(t, p)
}

func TestPingerContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	p, err := newPinger(ctx)
	require.NoError(t, err)
	go p.Run()
	time.Sleep(time.Second)
	assert.False(t, p.IsRunning())
}

func TestPingerLocalhost(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ip := "127.0.0.1"
	p, err := NewPinger(ctx)
	require.NoError(t, err)
	go p.Run()
	err = p.AddAddress(ip)
	require.NoError(t, err)
	c := time.NewTicker(time.Second)
	count := 0
	for {
		select {
		case <-c.C:
			if !p.IsRunning() {
				return
			}
			count++
			s := p.Statistics()
			assert.NotEmpty(t, s)
			l, ok := s[ip]
			assert.True(t, ok)
			logrus.Debug(l)
			assert.InDelta(t, count, l.PacketsRecv, 1)
			rtts := filterZeros(l.Rtts)
			assert.Equal(t, count, len(rtts))
		case <-ctx.Done():
			return
		}
	}
}

func TestPingerTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ip := "255.0.0.255"
	p, err := NewPinger(ctx)
	require.NoError(t, err)
	go p.Run()
	err = p.AddAddress(ip)
	require.NoError(t, err)
	c := time.NewTicker(time.Second)
	count := 0
	for {
		select {
		case <-c.C:
			count++
			if !p.IsRunning() {
				return
			}
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

func TestPingerReset(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 11*time.Second)
	defer cancel()
	ip1 := "255.0.0.255"
	ip2 := "127.0.0.1"
	p, err := NewPinger(ctx, ip1, ip2)
	require.NoError(t, err)
	l := logrus.New()
	l.SetLevel(logrus.DebugLevel)
	l.WithField("lib", "Pinger")
	p.SetLogger(l)
	go p.Run()
	to := time.After(5 * time.Second)
	c := time.NewTicker(time.Second)
	resetting := false
	resetDone := false
	count := 0
	for {
		select {
		case <-c.C:
			count++
			if !p.IsRunning() || resetting {
				logrus.Debug("Pinger not running or resetting")
				continue
			}
			if !resetting && count >= 5 && !resetDone {
				logrus.Debug("Re-adding ips")
				err = p.AddAddress(ip1)
				require.NoError(t, err)
				err = p.AddAddress(ip2)
				require.NoError(t, err)
				resetDone = true
				continue
			}
			if resetDone {
				s := p.Statistics()
				if !p.IsRunning() {
					return
				}
				logrus.Debug(s)

				l, ok := s[ip1]
				assert.True(t, ok)
				logrus.Debug(l)
				assert.Equal(t, 0, l.PacketsRecv)
				rtts := filterZeros(l.Rtts)
				assert.Equal(t, 0, len(rtts))
				l, ok = s[ip2]
				assert.True(t, ok)
				logrus.Debug(l)
				assert.InDelta(t, count-5, l.PacketsRecv, 2)
				rtts = filterZeros(l.Rtts)
				assert.InDelta(t, l.PacketsRecv, len(rtts), 2)
				continue
			}
			if len(p.Addresses()) < 2 {
				continue
			}
			s := p.Statistics()
			logrus.Debug(s)

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

		case <-to:
			logrus.Debug("Resetting Pinger")
			resetting = true
			p.Stop()
			err = p.RemoveAddress(ip1)
			require.NoError(t, err)
			err = p.RemoveAddress(ip2)
			require.NoError(t, err)
			go p.Run()
			resetting = false
		case <-ctx.Done():
			return
		}
	}

}

func TestTwoIPs(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ip1 := "255.0.0.255"
	ip2 := "127.0.0.1"
	p, err := NewPinger(ctx)
	require.NoError(t, err)
	go p.Run()
	err = p.AddAddress(ip1)
	require.NoError(t, err)
	err = p.AddAddress(ip2)
	require.NoError(t, err)
	c := time.NewTicker(time.Second)
	count := 0
	for {
		select {
		case <-c.C:
			count++
			s := p.Statistics()
			if !p.IsRunning() {
				return
			}
			assert.NotEmpty(t, s)
			logrus.Debug(s)

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

func TestPinger(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	p, err := NewPinger(ctx, "127.0.0.1", "255.0.0.255", "127.0.0.2")
	assert.NoError(t, err)
	if err != nil {
		t.FailNow()
	}
	p.SetLogger(logrus.StandardLogger())

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
			if !p.IsRunning() {
				return
			}
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	p, err := NewPinger(ctx, "127.0.0.1")
	assert.NoError(t, err)
	if err != nil {
		t.FailNow()
	}
	tk := time.NewTicker(time.Second)
	defer tk.Stop()
	count := 0
	go p.Run()
	for {
		select {
		case <-tk.C:
			count++
			s := p.Statistics()
			if !p.IsRunning() {
				return
			}
			ls, ok := s["127.0.0.1"]
			require.True(t, ok)
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
