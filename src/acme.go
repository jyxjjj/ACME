package main

import (
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/caddyserver/certmagic"
	"github.com/libdns/cloudflare"
	"github.com/mholt/acmez/v3"
	"github.com/mholt/acmez/v3/acme"
)

var (
	CADirectory string = certmagic.LetsEncryptStagingCA
	email       string
	domains     []string
	client      acmez.Client
	baseDir     string
	accountDir  string
	accountJson string
	accountKey  string
	domainDir   string
	domainCrt   string
	domainJson  string
	domainKey   string
)

func initACME() {
	email = os.Getenv("ACME_EMAIL")
	if email == "" || !isValidEmail(email) {
		Log.Fatalln("ACME_EMAIL is not set")
	}
	cfAPIToken := os.Getenv("CF_API_TOKEN")
	if cfAPIToken == "" {
		Log.Fatalln("CF_API_TOKEN is not set")
	}
	initDomains()
	initDirs()
	initACMEClient(
		initSolver(
			&cloudflare.Provider{
				APIToken: cfAPIToken,
			},
		),
	)
}

func initDirs() {
	// ./data/acme-staging-v02.api.letsencrypt.org/
	baseDir = "data/" + strings.TrimSuffix(strings.TrimPrefix(CADirectory, "https://"), "/directory")
	// ./data/acme-staging-v02.api.letsencrypt.org/accounts/{email}/
	accountDir = baseDir + "/accounts/" + email
	// ./data/acme-staging-v02.api.letsencrypt.org/accounts/{email}/account.json
	accountJson = accountDir + "/account.json"
	// ./data/acme-staging-v02.api.letsencrypt.org/accounts/{email}/account.key
	accountKey = accountDir + "/account.key"
	for domain := range domains {
		if isRootDomain(domains[domain]) {
			// ./data/acme-staging-v02.api.letsencrypt.org/certs/example.com/
			domainDir = baseDir + "/certs/" + domains[domain]
		}
	}
	// ./data/acme-staging-v02.api.letsencrypt.org/certs/example.com/cert.pem
	domainCrt = domainDir + "/cert.pem"
	// ./data/acme-staging-v02.api.letsencrypt.org/certs/example.com/cert.json
	domainJson = domainDir + "/cert.json"
	// ./data/acme-staging-v02.api.letsencrypt.org/certs/example.com/key.pem
	domainKey = domainDir + "/key.pem"
	// create dirs if not exist
	dirs := []string{baseDir, accountDir, domainDir}
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 02755); err != nil {
				Log.Fatalln("Failed to create directory:", dir, err)
			}
		}
	}
}

func initDomains() {
	domain := os.Getenv("CERT_DOMAIN")
	if domain == "" {
		Log.Fatalln("CERT_DOMAIN is not set")
	}
	split := strings.SplitSeq(domain, ",")
	for domain := range split {
		domains = append(domains, domain)
	}
}

func initSolver(cloudflareSolver certmagic.DNSProvider) *certmagic.DNS01Solver {
	return &certmagic.DNS01Solver{
		DNSManager: certmagic.DNSManager{
			DNSProvider:        cloudflareSolver,
			PropagationDelay:   15 * time.Second,
			PropagationTimeout: 5 * time.Minute,
			Resolvers: []string{
				"127.0.0.53:53",
			},
		},
	}
}

func initACMEClient(solver *certmagic.DNS01Solver) {
	client = acmez.Client{
		Client: &acme.Client{
			Directory:  CADirectory,
			UserAgent:  "DESMG ACME Service",
			HTTPClient: http.DefaultClient,
			Logger:     slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})),
		},
		ChallengeSolvers: map[string]acmez.Solver{
			acme.ChallengeTypeDNS01:     solver,
			acme.ChallengeTypeHTTP01:    nil,
			acme.ChallengeTypeTLSALPN01: nil,
		},
	}
}

// main certificate management workflow
func manageCertificates() error {
	acc, err := getOrRegisterAccount()
	if err != nil {
		return err
	}
	Log.Println("[ACME] Using Account:", acc.Location)
	err = newOrRenewCert()
	if err != nil {
		return err
	}
	return nil
}
