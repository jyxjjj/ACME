package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/caddyserver/certmagic"
	"github.com/gin-gonic/gin"
	"github.com/libdns/cloudflare"
)

var r = gin.Default()
var addr, port string
var domain string

func init() {
	// gin
	addr = os.Getenv("ADDR")
	port = os.Getenv("PORT")
	if addr == "" {
		addr = "0.0.0.0"
	}
	if port == "" {
		port = "9504"
	}
	r.RemoveExtraSlash = true
	r.NoRoute(func(c *gin.Context) {
		c.String(404, "Not Found")
	})
	r.GET("/cert", authVerify, getCert)
	r.GET("/key", authVerify, getKey)

	// acme
	email := os.Getenv("ACME_EMAIL")
	if email == "" {
		log.Fatal("ACME_EMAIL is not set")
	}
	domain = os.Getenv("CERT_DOMAIN")
	if domain == "" {
		log.Fatal("CERT_DOMAIN is not set")
	}
	cfAPIToken := os.Getenv("CF_API_TOKEN")
	if cfAPIToken == "" {
		log.Fatal("CF_API_TOKEN is not set")
	}
	cloudflareSolver := &cloudflare.Provider{
		APIToken: cfAPIToken,
	}
	certmagic.Default.Storage = &certmagic.FileStorage{Path: "./certs"}
	certmagic.DefaultKeyGenerator.KeyType = certmagic.P256
	certmagic.DefaultACME.Email = email
	certmagic.DefaultACME.Agreed = true
	certmagic.DefaultACME.CA = certmagic.LetsEncryptStagingCA
	certmagic.DefaultACME.DisableHTTPChallenge = true
	certmagic.DefaultACME.DisableTLSALPNChallenge = true
	certmagic.DefaultACME.Profile = "tlsserver"
	certmagic.DefaultACME.DNS01Solver = &certmagic.DNS01Solver{
		DNSManager: certmagic.DNSManager{
			DNSProvider:        cloudflareSolver,
			PropagationDelay:   15 * time.Second,
			PropagationTimeout: 120 * time.Second,
			Resolvers: []string{
				"127.0.0.53:53",
			},
		},
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		d := &net.Dialer{Timeout: 10 * time.Second}
		return d.DialContext(ctx, "tcp4", addr)
	}
	http.DefaultClient.Transport = transport
}

func main() {
	log.Println(strings.Repeat("=", 32))
	log.Println("DESMG ACME Service")
	log.Println("Copyright (C) 2025")
	log.Println("DESMG All rights reserved.")
	log.Println("This software was licensed under the GNU Affero General Public License v3.0 only.")
	log.Println(strings.Repeat("=", 32))
	domains := []string{}
	split := strings.SplitSeq(domain, ",")
	for domain := range split {
		domains = append(domains, domain)
	}
	err := certmagic.ManageSync(context.Background(), domains)
	if err != nil {
		log.Fatalf("Failed to ensure certificate %v", err)
	}
	r.Run(addr + ":" + port)
}
