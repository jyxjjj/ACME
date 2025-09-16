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
	"slices"

	"github.com/mholt/acmez/v3/acme"
)

func getOrRegisterAccount() (*acme.Account, error) {
	ctx := context.Background()
	wantMail := "mailto:" + email
	if _, err := os.Stat(accountJson); os.IsNotExist(err) {
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
	} else {
		Log.Println("[ACME] Found existing account file:", accountJson)
		acc, err := os.ReadFile(accountJson)
		if err != nil {
			return nil, fmt.Errorf("error while reading account file: %v", err)
		}
		key, err := os.ReadFile(accountKey)
		if err != nil {
			return nil, fmt.Errorf("error while reading account key file: %v", err)
		}
		block, _ := pem.Decode(key)
		if block == nil || block.Type != "EC PRIVATE KEY" {
			return nil, fmt.Errorf("failed to decode PEM block containing private key")
		}
		accountPrivateKey, err := x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("error while parsing account private key: %v", err)
		}
		var account acme.Account
		err = json.Unmarshal(acc, &account)
		if err != nil {
			return nil, fmt.Errorf("error while unmarshaling account file: %v", err)
		}
		account.PrivateKey = accountPrivateKey
		account, err = client.GetAccount(ctx, account)
		if err != nil {
			return nil, fmt.Errorf("error while getting account: %v", err)
		}
		if account.Status == acme.StatusDeactivated || account.Status == acme.StatusRevoked {
			Log.Println("[ACME] Account is deactivated or revoked. Re-registering...")
			os.Remove(accountJson)
			os.Remove(accountKey)
			return getOrRegisterAccount()
		}
		if !slices.Equal(account.Contact, []string{wantMail}) {
			account.Contact = []string{wantMail}
			Log.Println("[ACME] Updating account contact to:", account.Contact)
			updatedAccount, err := client.UpdateAccount(ctx, account)
			if err != nil {
				return nil, fmt.Errorf("error while updating account: %v", err)
			}
			data, err := json.Marshal(updatedAccount)
			if err == nil {
				_ = os.WriteFile(accountJson, data, 0600)
			}
			account = updatedAccount
		}
		Log.Println("[ACME] Using existing account with email:", email)
		return &account, nil
	}
}
