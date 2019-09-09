package ping

import (
	"math"
	"net"
	"sync"
	"time"

	"github.com/digineo/go-ping"
	"github.com/sirupsen/logrus"
)

type history struct {
	received int
	lost     int
	results  []time.Duration // ring, start index = .received%len
	mtx      sync.RWMutex
}

type destination struct {
	host    string
	remote  *net.IPAddr
	display string
	*history
}

func (u *destination) ping(pinger *ping.Pinger, timeout time.Duration) {
	rtt, err := pinger.Ping(u.remote, timeout)
	if err != nil {
		logrus.Debugf("%s: %v", u.host, err)
	}
	u.addResult(rtt, err)
}

func (s *history) addResult(rtt time.Duration, err error) {
	s.mtx.Lock()

	// TODO : What if we reach max of int
	if err == nil {
		s.received++
		// On windows there may be rtt 0 caused by bad time resolution
		if rtt == 0 {
			rtt = 1
		}
	} else {
		s.lost++
	}
	switch len(s.results) {
	case 0:
	case 1:
		s.results[0] = rtt
	default:
		s.results = append([]time.Duration{rtt}, s.results[:len(s.results)-1]...)
	}
	s.mtx.Unlock()
}

func (s *history) compute() (st Statistics) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	st.PacketsSent = s.received + s.lost
	st.PacketsRecv = s.received
	if s.received == 0 {
		if s.lost > 0 {
			st.PacketLoss = 100
			st.Rtts = make([]time.Duration, len(s.results))
			copy(st.Rtts, s.results)
		}
		return
	}

	st.PacketLoss = float64(s.lost) / float64(s.received+s.lost) * 100
	st.MinRtt, st.MaxRtt = s.results[0], s.results[0]

	total := time.Duration(0)
	count := 0
	for _, rtt := range s.results {
		if rtt < st.MinRtt && rtt != 0 {
			st.MinRtt = rtt
		}
		if rtt > st.MaxRtt {
			st.MaxRtt = rtt
		}
		if rtt != 0 {
			count++
		}
		total += rtt
	}

	if count == 0 {
		count = 1
	}

	st.AvgRtt = time.Duration(float64(total) / float64(count))

	stddevNum := float64(0)
	for _, rtt := range s.results {
		stddevNum += math.Pow(float64(rtt-st.AvgRtt), 2)
	}
	st.StdDevRtt = time.Duration(math.Sqrt(stddevNum / float64(count)))
	st.Rtts = make([]time.Duration, len(s.results))
	copy(st.Rtts, s.results)
	return
}
