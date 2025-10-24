package main

import (
	"os"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
)

func init() {
	// Validate essential auth-related env vars at startup so failures surface early
	orgName := os.Getenv("CF_ZT_ORG_NAME")
	policyAUD := os.Getenv("CF_ZT_AUD")
	if orgName == "" || policyAUD == "" {
		Log.Fatalln("CF_ZT_ORG_NAME or CF_ZT_AUD is not set; auth cannot function")
	}
}

func authVerify(c *gin.Context) {
	token := c.GetHeader("Cf-Access-Jwt-Assertion")
	if token == "" {
		jsonResponse(c, ErrUnauthorized, nil)
		c.Abort()
		return
	}
	orgName := strings.ToLower(os.Getenv("CF_ZT_ORG_NAME"))
	policyAUD := os.Getenv("CF_ZT_AUD")
	if orgName == "" || policyAUD == "" {
		jsonResponse(c, ErrServerMisconfig, nil)
		c.Abort()
		return
	}
	certsURL := "https://" + orgName + ".cloudflareaccess.com/cdn-cgi/access/certs"
	config := &oidc.Config{
		ClientID: policyAUD,
	}
	ctx := c.Request.Context()
	keySet := oidc.NewRemoteKeySet(ctx, certsURL)
	verifier := oidc.NewVerifier("https://"+orgName+".cloudflareaccess.com", keySet, config)
	_, err := verifier.Verify(ctx, token)
	if err != nil {
		jsonResponse(c, ErrUnauthorized, nil)
		c.Abort()
		return
	}
	c.Next()
}
