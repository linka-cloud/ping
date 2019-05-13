package ping

import "time"

type options struct {
	timeout         time.Duration
	interval        time.Duration
	payloadSize     uint
	statBufferSize  uint
	bind4           string
	bind6           string
	dests           []*destination
	resolverTimeout time.Duration
}

var defaultOptions = options{
	timeout:         500 * time.Millisecond,
	interval:        time.Second,
	bind4:           "0.0.0.0",
	bind6:           "::",
	payloadSize:     56,
	statBufferSize:  10,
	resolverTimeout: time.Second,
}
