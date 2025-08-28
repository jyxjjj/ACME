package main

import (
	"context"
	"net"
	"net/http"
)

func initHTTPClient() {
	http.DefaultClient.Transport = getIPv4OnlyTransport()
}

func getIPv4OnlyTransport() *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	originalDialContext := transport.DialContext
	transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return originalDialContext(ctx, "tcp4", addr)
	}
	return transport
}
