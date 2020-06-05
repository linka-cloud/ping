# Ping

Ping is based on [github.com/digineo/go-ping](https://github.com/digineo/go-ping), 
more precisely on [github.com/digineo/go-ping/cmd/multiping](https://github.com/digineo/go-ping/tree/master/cmd/multiping).

It allow to ping multiple destinations.

Example : 

```go
package main

import (
    "context"
    "time"

    "github.com/sirupsen/logrus"
    
    "gitlab.bertha.cloud/partitio/isi/ping"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
    defer cancel()
    p, err := ping.NewPinger(ctx, "127.0.0.1", "255.0.0.1", "192.168.12.1")
    if err != nil {
        logrus.Fatal(err)
    }
    defer p.Close()

    go p.Run()

    t := time.NewTicker(time.Second)
    for {
        select {
        case <-t.C:
            sts := p.Statistics()
            for _, s := range sts {
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
        case <-ctx.Done():
            return
        }
    }
}
```

The pinger :

```go
type Pinger interface {
    // deprecated: removed udp support
    Privileged() bool

    // Addresses returns the list of the destinations host
    Addresses() []string
    // IPAddresses returns the list of the destinations addresses
    IPAddresses() []net.IPAddr

    // AddAddress add a destination to the pinger
    AddAddress(a string) error
    // Remove Address remove a destination from the pinger
    RemoveAddress(a string) error

    // Run start the pinger. It fails silently if the pinger is already running
    Run()
    // Stop stops the pinger. It fails silently if the pinger is already stopped
    Stop()

    // IsRunning returns the state of the pinger
    IsRunning() bool

    // Statistics returns the a map address ping Statistics
    Statistics() map[string]Statistics

    // Close closes the connection. It should be call deferred right after the creation of the pinger
    Close()
}
```

The Statistics :

```go
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
    // 0 means nothing was received
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

```

