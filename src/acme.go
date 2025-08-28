package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
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
			if err := os.MkdirAll(dir, 2755); err != nil {
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
	acc, err := getOrRegisterAccount(email)
	if err != nil {
		return err
	}
	Log.Println("[ACME] Using Account:", acc.Location)
	return nil
}

func getOrRegisterAccount(email string) (*acme.Account, error) {
	ctx := context.Background()
	wantMail := "mailto:" + email
	// data, err := os.ReadFile(accountJson)
	// if err == nil {
	// 	Log.Println("[ACME] Found existing account file:", accountJson)
	// 	var acc acme.Account
	// 	err := json.Unmarshal(data, &acc)
	// 	if err == nil {
	// 		Log.Println("[ACME] Loading existing account:", acc.Location)
	// 		found := slices.Contains(acc.Contact, wantMail)
	// 		if !found {
	// 			Log.Println("[ACME] Updating account email to:", email)
	// 			acc.Contact = []string{wantMail}
	// 			updated, err := client.UpdateAccount(ctx, acc)
	// 			if err != nil {
	// 				return nil, err
	// 			}
	// 			if data, err := json.Marshal(updated); err == nil {
	// 				_ = os.WriteFile(accountJson, data, 0600)
	// 			}
	// 			return &updated, nil
	// 		}
	// 		return &acc, nil
	// 	}
	// }
	Log.Println("[ACME] Registering Account with email:", email)
	Log.Println("[ACME] Generating new account private key...")
	accountPrivateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("error while generating account key: %v", err)
	}
	Log.Println("[ACME] Saving new account private key to:", accountKey)
	keyBytes, err := x509.MarshalECPrivateKey(accountPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("error while marshaling account key: %v", err)
	}
	pemBlock := &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyBytes,
	}
	pemData := pem.EncodeToMemory(pemBlock)
	err = os.WriteFile(accountKey, []byte(pemData), 0600)
	if err != nil {
		return nil, fmt.Errorf("error while saving account key: %v", err)
	}
	account := acme.Account{
		Contact:              []string{wantMail},
		TermsOfServiceAgreed: true,
		PrivateKey:           accountPrivateKey,
	}
	account, err = client.NewAccount(ctx, account)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(account)
	if err == nil {
		_ = os.WriteFile(accountJson, data, 0600)
	}
	return &account, nil
}
