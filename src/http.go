package main

import (
	"context"
	"net"
	"net/http"
)

const BASE_UA = "DESMG"
const APP_UA = "ACME Service"

type clientTransport struct {
	transport http.RoundTripper
}

func (base *clientTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", BASE_UA+" "+APP_UA)
	return base.transport.RoundTrip(req)
}

func initHTTPClient() {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext(ctx, "tcp4", addr)
	}
	transport.Proxy = http.ProxyFromEnvironment
	transport.TLSHandshakeTimeout = 10 * time.Second
	transport.IdleConnTimeout = 90 * time.Second
	http.DefaultClient.Timeout = 60 * time.Second
	http.DefaultClient.Transport = &clientTransport{transport: transport}
}
