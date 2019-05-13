package ping

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

func resolve(addr string, timeout time.Duration) (net.IPAddr, error) {
	if strings.ContainsRune(addr, '%') {
		ipaddr, err := net.ResolveIPAddr("ip", addr)
		if err != nil {
			return net.IPAddr{}, err
		}
		return *ipaddr, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, addr)
	if err != nil {
		return net.IPAddr{}, err
	}
	if len(addrs) < 1 {
		return net.IPAddr{}, fmt.Errorf("%s : no ip found", addr)
	}
	return addrs[0], nil
}
