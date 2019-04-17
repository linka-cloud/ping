// Pinger incapsulate a [go-ping Pinger](https://github.com/sparrc/go-ping) to provide multiple address support.
//
//
// func main() {
//    // Initialize context
//    ctx, cancel := context.WithCancel(context.Background())
//    // Cancel context after 10 seconds
//    go func() {
//        time.Sleep(10 * time.Second)
//        cancel()
//    }()
//    // Create a Pinger
//    p, err := NewPinger(ctx, "127.0.0.1", "192.168.255.22")
//    if err != nil {
//        panic(err)
//    }
//    // Start the pinger in background
//    go p.Run()
//
//    // Add a new address while running after 3 seconds
//    go func() {
//        time.Sleep(3 * time.Second)
//        if err := p.AddAddress("192.168.52.112"); err != nil {
//            panic(err)
//        }
//    }()
//    // Remove an address while running after 5 seconds
//    go func() {
//        time.Sleep(5 * time.Second)
//        if err := p.RemoveAddress("127.0.0.1"); err != nil {
//            panic(err)
//        }
//    }()
//    // Create a ticker to get the status every second
//    t := time.NewTicker(time.Second)
//    for {
//        select {
//        case <-t.C:
//            // Get the status
//            fmt.Println(p.Status())
//        case <-ctx.Done():
//            fmt.Println(ctx.Err())
//            return
//        }
//    }
//}
package ping
