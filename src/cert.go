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

	"github.com/mholt/acmez/v3/acme"
)

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
	err = os.WriteFile(domainKey, []byte(pemData), 0600)
	if err != nil {
		return fmt.Errorf("error saving domain private key: %v", err)
	}
	certs, err := client.ObtainCertificateForSANs(ctx, account, domainPrivateKey, domains)
	if err != nil {
		return fmt.Errorf("error obtaining certificate: %v", err)
	}
	for _, cert := range certs {
		certificates := strings.Split(string(cert.ChainPEM), "\n\n")
		intermediateCA := certificates[1]
		intermediateCABlock, _ := pem.Decode([]byte(intermediateCA))
		if intermediateCABlock == nil {
			return fmt.Errorf("error decoding intermediate CA certificate: no PEM block found")
		}
		intermediateCABlockParsed, err := x509.ParseCertificate(intermediateCABlock.Bytes)
		if err != nil {
			return fmt.Errorf("error parsing intermediate CA certificate: %v", err)
		}
		if intermediateCABlockParsed.SignatureAlgorithm != x509.ECDSAWithSHA384 {
			continue
		}
		Log.Println("[ACME] Saving certificate to:", domainCrt)
		err = os.WriteFile(domainCrt, cert.ChainPEM, 0644)
		if err != nil {
			return fmt.Errorf("error saving certificate pem: %v", err)
		}
		json, err := json.Marshal(cert)
		if err != nil {
			return fmt.Errorf("error marshaling certificate json: %v", err)
		}
		err = os.WriteFile(domainJson, json, 0644)
		if err != nil {
			return fmt.Errorf("error saving certificate json: %v", err)
		}
	}
	Log.Println("[ACME] Successfully obtained and saved new certificate for domains:", strings.Join(domains, ", "))
	return nil
}
func renewCert(ctx context.Context) error {
	Log.Println("[ACME] Found existing certificate, checking if renewal is needed for domains:", strings.Join(domains, ", "))
	jsonBytes, err := os.ReadFile(domainJson)
	if err != nil {
		return err
	}
	var cert *acme.Certificate
	err = json.Unmarshal(jsonBytes, &cert)
	if err != nil {
		return err
	}
	return nil
	// TODO check time and renew if needed
	return newCert(ctx)
}
