package main

import (
	"net/http"
	"os"
	"runtime"
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
	account     acme.Account
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
	// ./data/acme-staging-v02.api.letsencrypt.org/certs/example.com/
	domainDir = baseDir + "/certs/" + domains[0]
	// ./data/acme-staging-v02.api.letsencrypt.org/certs/example.com/cert.pem
	domainCrt = domainDir + "/cert.crt"
	// ./data/acme-staging-v02.api.letsencrypt.org/certs/example.com/cert.json
	domainJson = domainDir + "/cert.json"
	// ./data/acme-staging-v02.api.letsencrypt.org/certs/example.com/key.pem
	domainKey = domainDir + "/priv.key"
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
	for d := range split {
		domains = append(domains, d)
	}
	if len(domains) == 0 {
		Log.Fatalln("No valid domain found in CERT_DOMAIN")
	}
	Log.Println("[Manager] Managing certificate for domains:", strings.Join(domains, ", "))
}

func initSolver(cloudflareSolver certmagic.DNSProvider) *certmagic.DNS01Solver {
	return &certmagic.DNS01Solver{
		DNSManager: certmagic.DNSManager{
			DNSProvider:        cloudflareSolver,
			PropagationDelay:   3 * time.Second,
			PropagationTimeout: 2 * time.Minute,
			Resolvers: func() []string {
				if runtime.GOOS == "linux" {
					return []string{
						"127.0.0.53:53",
					}
				}
				return []string{
					"127.0.0.1:53",
					"1.1.1.1:53",
					"1.0.0.1:53",
					"8.8.8.8:53",
					"8.8.4.4:53",
					"223.5.5.5:53",
					"223.6.6.6:53",
				}
			}(),
		},
	}
}

func initACMEClient(solver *certmagic.DNS01Solver) {
	client = acmez.Client{
		Client: &acme.Client{
			Directory:  CADirectory,
			HTTPClient: http.DefaultClient,
			Logger:     getLogrusSLogProxy(),
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
	Log.Println("[Manager] ACME Directory:", CADirectory)
	err := getOrRegisterAccount()
	if err != nil {
		return err
	}
	Log.Println("[ACME] Using Account:", account.Location)
	err = newOrRenewCert()
	if err != nil {
		return err
	}
	return nil
}
