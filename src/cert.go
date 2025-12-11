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
	"os"
	"strings"
	"sync/atomic"
	"time"
)

var selectedRenewalTime atomic.Value // stores time.Time
var selectedInProgress atomic.Int32

func newOrRenewCert() error {
	ctx := context.Background()
	if _, err := os.Stat(domainJson); os.IsNotExist(err) {
		return newCert(ctx)
	} else {
		return renewCert(ctx)
	}
}

func newCert(ctx context.Context) error {
	Log.Println("[ACME] No existing certificate found, requesting new one for domains:", strings.Join(domains, ", "))
	domainPrivateKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return fmt.Errorf("error generating domain private key: %v", err)
	}
	Log.Println("[ACME] Saving domain private key to:", domainKey)
	keyBytes, err := x509.MarshalECPrivateKey(domainPrivateKey)
	if err != nil {
		return fmt.Errorf("error marshaling domain private key: %v", err)
	}
	pemBlock := &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyBytes,
	}
	pemData := pem.EncodeToMemory(pemBlock)
	if err := os.WriteFile(domainKey, []byte(pemData), 0600); err != nil {
		return fmt.Errorf("error saving domain private key: %v", err)
	}
	certs, err := client.ObtainCertificateForSANs(ctx, account, domainPrivateKey, domains)
	if err != nil {
		return fmt.Errorf("error obtaining certificate: %v", err)
	}
	for _, cert := range certs {
		// Normalize PEM: replace double newlines with single to ease splitting
		pemData := []byte(strings.ReplaceAll(string(cert.ChainPEM), "\n\n", "\n"))
		// Parse all PEM blocks
		var blocks [][]byte
		rest := pemData
		for {
			var block *pem.Block
			block, rest = pem.Decode(rest)
			if block == nil {
				break
			}
			blocks = append(blocks, block.Bytes)
		}
		if len(blocks) == 0 {
			return fmt.Errorf("no PEM blocks found in certificate chain")
		}
		// Expect first block to be leaf, second (if present) intermediate
		var intermediateCert *x509.Certificate
		if len(blocks) >= 2 {
			intermediate, err := x509.ParseCertificate(blocks[1])
			if err != nil {
				return fmt.Errorf("error parsing intermediate CA certificate: %v", err)
			}
			intermediateCert = intermediate
		}
		// If there is an intermediate certificate, require it to be ECDSA (per project constraint)
		if intermediateCert != nil {
			if intermediateCert.PublicKeyAlgorithm != x509.ECDSA {
				return fmt.Errorf("intermediate CA public key algorithm is not ECDSA; found: %v", intermediateCert.PublicKeyAlgorithm)
			}
		}
		Log.Println("[ACME] Saving certificate to:", domainCrt)
		if err := os.WriteFile(domainCrt, cert.ChainPEM, 0644); err != nil {
			return fmt.Errorf("error saving certificate pem: %v", err)
		}
		jsonData, err := json.Marshal(cert)
		if err != nil {
			return fmt.Errorf("error marshaling certificate json: %v", err)
		}
		if err := os.WriteFile(domainJson, jsonData, 0644); err != nil {
			return fmt.Errorf("error saving certificate json: %v", err)
		}
		// register upstream suggested renewal time (if provided)
		_ = registerSelectedRenewalTimeFromJSON(jsonData)
	}
	Log.Println("[ACME] Successfully obtained and saved new certificate for domains:", strings.Join(domains, ", "))
	return nil
}
func renewCert(ctx context.Context) error {
	Log.Println("[ACME] Found existing certificate, checking if renewal is needed for domains:", strings.Join(domains, ", "))
	// Prefer to use upstream-provided renewal recommendation from domainJson
	jsonBytes, err := os.ReadFile(domainJson)
	if err == nil {
		var payload map[string]any
		if err := json.Unmarshal(jsonBytes, &payload); err == nil {
			if riRaw, ok := payload["renewal_info"]; ok {
				if ri, ok := riRaw.(map[string]any); ok {
					// Prefer _selectedTime, then suggestedWindow.start
					var selectedStr string
					if s, ok := ri["_selectedTime"].(string); ok && s != "" {
						selectedStr = s
					} else if swRaw, ok := ri["suggestedWindow"]; ok {
						if sw, ok := swRaw.(map[string]any); ok {
							if start, ok := sw["start"].(string); ok && start != "" {
								selectedStr = start
							}
						}
					}
					if selectedStr != "" {
						// Try parsing with RFC3339 and RFC3339Nano. We intentionally
						// use a short declaration for `err` here so the parse error
						// is scoped to this parsing block only and does not
						// overwrite any outer `err` that may be used later in the
						// function. This keeps error handling local to the parse
						// branch and avoids accidentally shadowing outer state.
						var selTime time.Time
						selTime, err := time.Parse(time.RFC3339, selectedStr)
						if err != nil {
							selTime, err = time.Parse(time.RFC3339Nano, selectedStr)
						}
						if err == nil {
							now := time.Now()
							if now.Before(selTime) {
								Log.Println("[ACME] Renewal scheduled by upstream at", selTime, "; skipping until then")
								return nil
							}
							// else fallthrough to perform renewal
							Log.Println("[ACME] Upstream-recommended renewal time reached (", selTime, "), proceeding to renew")
							return newCert(ctx)
						}
						Log.Println("[ACME] Could not parse upstream selectedTime:", selectedStr, "err:", err)
					}
				}
			}
		} else {
			Log.Println("[ACME] Warning: failed to parse", domainJson, "err:", err)
		}
	} else {
		Log.Println("[ACME] Warning: could not read domain json:", err)
	}

	// Fallback: parse local cert and use 30-day rule
	pemBytes, err := os.ReadFile(domainCrt)
	if err != nil {
		return fmt.Errorf("error reading certificate pem: %v", err)
	}
	// Decode first PEM block (leaf)
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return fmt.Errorf("no PEM block found in certificate file")
	}
	certParsed, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("error parsing certificate: %v", err)
	}
	// Renew if less than 30 days remaining
	remaining := time.Until(certParsed.NotAfter)
	if remaining <= 0 {
		Log.Println("[ACME] Certificate already expired, requesting new one...")
		return newCert(ctx)
	}
	if remaining <= (30 * 24 * time.Hour) {
		Log.Println("[ACME] Certificate expiring within 30 days, renewing...")
		return newCert(ctx)
	}
	Log.Println("[ACME] Certificate not due for renewal. Remaining:", remaining)
	return nil
}

// read selectedTime or suggestedWindow from domainJson and store into selectedRenewalTime
func loadSelectedRenewalTime() error {
	jsonBytes, err := os.ReadFile(domainJson)
	if err != nil {
		return err
	}
	return registerSelectedRenewalTimeFromJSON(jsonBytes)
}

func registerSelectedRenewalTimeFromJSON(jsonBytes []byte) error {
	var payload map[string]any
	if err := json.Unmarshal(jsonBytes, &payload); err != nil {
		return err
	}
	riRaw, ok := payload["renewal_info"]
	if !ok {
		return nil
	}
	ri, ok := riRaw.(map[string]any)
	if !ok {
		return nil
	}
	var selectedStr string
	if s, ok := ri["_selectedTime"].(string); ok && s != "" {
		selectedStr = s
	} else if swRaw, ok := ri["suggestedWindow"]; ok {
		if sw, ok := swRaw.(map[string]any); ok {
			if start, ok := sw["start"].(string); ok && start != "" {
				selectedStr = start
			}
		}
	}
	if selectedStr == "" {
		return nil
	}
	// parse time
	// We use a short declaration for `err` to ensure any parse error is
	// localized here. This function returns the parse error immediately, so
	// scoping `err` to this block is safe and clearer for reviewers.
	var selTime time.Time
	selTime, err := time.Parse(time.RFC3339, selectedStr)
	if err != nil {
		selTime, err = time.Parse(time.RFC3339Nano, selectedStr)
	}
	if err != nil {
		return err
	}
	selectedRenewalTime.Store(selTime)
	return nil
}
