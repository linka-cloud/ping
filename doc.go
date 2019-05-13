//
//# Ping
//
//Ping is based on [github.com/digineo/go-ping](https://github.com/digineo/go-ping),
//more precisely on [github.com/digineo/go-ping/cmd/multiping](https://github.com/digineo/go-ping/tree/master/cmd/multiping)
//
//```go
//package main
//
//import (
//    "context"
//    "time"
//
//    "github.com/sirupsen/logrus"
//
//    "gitlab.bertha.cloud/partitio/isi/ping"
//)
//
//func main() {
//    ctx, cancel := context.WithCancel(context.Background())
//    go func() {
//        time.Sleep(5 * time.Second)
//        cancel()
//    }()
//    p, err := ping.NewPinger(ctx, "127.0.0.1", "255.0.0.1", "192.168.12.1")
//    if err != nil {
//        logrus.Fatal(err)
//    }
//    defer p.Close()
//
//    go p.Run()
//
//    t := time.NewTicker(time.Second)
//    for {
//        select {
//        case <-t.C:
//            sts := p.Statistics()
//            for _, s := range sts {
//                logrus.WithFields(logrus.Fields{
//                    "host":     s.Addr,
//                    "address":  s.IPAddr,
//                    "sent":     s.PacketsSent,
//                    "lost":     s.PacketLoss,
//                    "received": s.PacketsRecv,
//                    "min":      s.MinRtt,
//                    "max":      s.MaxRtt,
//                    "mean":     s.AvgRtt,
//                    "rtts":     s.Rtts,
//                }).Info()
//            }
//        case <-ctx.Done():
//            return
//        }
//    }
//}
//```
package ping
