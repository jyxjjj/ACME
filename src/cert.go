package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
)

func newOrRenewCert() error {
	ctx := context.Background()
	if _, err := os.Stat(domainCrt); os.IsNotExist(err) {
		return newCert(ctx)
	} else {
		return renewCert(ctx)
	}
}

func newCert(ctx context.Context) error {
	Log.Println("[ACME] No existing certificate found, requesting new one for domains:", strings.Join(domains, ", "))
	domainPrivateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
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
	certPem := []byte{}
	for _, cert := range certs {
		certPem = append(certPem, cert.ChainPEM...)
		certPem = append(certPem, cert.CA...)
	}
	Log.Println("[ACME] Saving certificate to:", domainCrt)
	err = os.WriteFile(domainCrt, certPem, 0644)
	if err != nil {
		return fmt.Errorf("error saving certificate: %v", err)
	}
	Log.Println("[ACME] Successfully obtained and saved new certificate for domains:", strings.Join(domains, ", "))
	return nil
}
func renewCert(ctx context.Context) error {
	Log.Println("[ACME] Found existing certificate, checking if renewal is needed for domains:", strings.Join(domains, ", "))
	cert, err := os.ReadFile(domainCrt)
	if err != nil {
		return err
	}
	Log.Println("[ACME] Current certificate:\n", string(cert))
	return nil
}
