package ping

import (
	"net"
	"time"

	"github.com/sirupsen/logrus"
)

type Statistics struct {
	// PacketsRecv is the number of packets received.
	PacketsRecv int

	// PacketsSent is the number of packets sent.
	PacketsSent int

	// PacketLoss is the percentage of packets lost.
	PacketLoss float64

	// IPAddr is the address of the host being pinged.
	IPAddr net.IPAddr

	// Addr is the string address of the host being pinged.
	Addr string

	// Rtts is the last 10 round-trip times sent via this pinger.
	// 0 means timeout
	Rtts []time.Duration

	// MinRtt is the minimum round-trip time sent via this pinger.
	MinRtt time.Duration

	// MaxRtt is the maximum round-trip time sent via this pinger.
	MaxRtt time.Duration

	// AvgRtt is the average round-trip time sent via this pinger.
	AvgRtt time.Duration

	// StdDevRtt is the standard deviation of the round-trip times sent via
	// this pinger.
	StdDevRtt time.Duration
}

func logStats(s Statistics) {
	logrus.WithFields(logrus.Fields{
		"host":     s.Addr,
		"address":  s.IPAddr,
		"sent":     s.PacketsSent,
		"lost":     s.PacketLoss,
		"received": s.PacketsRecv,
		"min":      s.MinRtt,
		"max":      s.MaxRtt,
		"mean":     s.AvgRtt,
		"rtts":     s.Rtts,
	}).Info()
}
