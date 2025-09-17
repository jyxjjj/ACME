package main

import (
	"context"
	"net"
	"net/http"
)

type clientTransport struct {
	baseTransport http.RoundTripper
}

func initHTTPClient() {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	originalDialContext := transport.DialContext
	transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return originalDialContext(ctx, "tcp4", addr)
	}
	http.DefaultClient.Transport = &clientTransport{baseTransport: transport}
}

func (base *clientTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", "DESMG ACME Service")
	return base.baseTransport.RoundTrip(req)
}
