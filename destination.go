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
	s.results[(s.received+s.lost)%len(s.results)] = rtt
	if err == nil {
		s.received++
	} else {
		s.lost++
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
			st.Rtts = make([]time.Duration, 10)
			copy(st.Rtts, s.results)
		}
		return
	}

	collection := s.results[:]
	st.PacketsSent = s.received + s.lost
	size := len(s.results)

	// we don't yet have filled the buffer
	if s.received <= size {
		collection = s.results[:s.received]
		size = s.received
	}

	st.PacketLoss = float64(s.lost) / float64(s.received+s.lost) * 100
	st.MinRtt, st.MaxRtt = collection[0], collection[0]

	total := time.Duration(0)
	for _, rtt := range collection {
		if rtt < st.MinRtt {
			st.MinRtt = rtt
		}
		if rtt > st.MaxRtt {
			st.MaxRtt = rtt
		}
		total += rtt
	}

	st.AvgRtt = time.Duration(float64(total) / float64(size))

	stddevNum := float64(0)
	for _, rtt := range collection {
		stddevNum += math.Pow(float64(rtt-st.AvgRtt), 2)
	}
	st.StdDevRtt = time.Duration(math.Sqrt(stddevNum / float64(size)))
	st.Rtts = make([]time.Duration, 10)
	copy(st.Rtts, s.results)
	return
}
