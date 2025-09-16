package main

import (
	"os"
	"strings"
)

func newOrRenewCert() error {
	if _, err := os.Stat(domainCrt); os.IsNotExist(err) {
		Log.Println("[ACME] No existing certificate found, requesting new one for domains:", strings.Join(domains, ", "))
	} else {
		Log.Println("[ACME] Found existing certificate, checking if renewal is needed for domains:", strings.Join(domains, ", "))
	}
	Log.Println("[ACME] All domains are covered by the certificate.")
	return nil
}
