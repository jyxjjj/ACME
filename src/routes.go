package main

import (
	"os"

	"github.com/gin-gonic/gin"
)

func getCert(c *gin.Context) {
	domain := c.Query("domain")
	if domain == "" {
		jsonResponse(c, ErrDomainRequired, nil)
		return
	}

	// TODO: path is wrong
	certPath := "./certs/" + domain + "/cert.crt"
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		jsonResponse(c, ErrFileNotFound, nil)
		return
	}
	content, err := os.ReadFile(certPath)
	if err != nil {
		jsonResponse(c, ErrReadFileFailed, nil)
		return
	}
	jsonResponse(c, ErrSuccess, string(content))
}

func getKey(c *gin.Context) {
	domain := c.Query("domain")
	if domain == "" {
		jsonResponse(c, ErrDomainRequired, nil)
		return
	}

	// TODO: path is wrong
	keyPath := "./certs/" + domain + "/priv.key"
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		jsonResponse(c, ErrFileNotFound, nil)
		return
	}
	content, err := os.ReadFile(keyPath)
	if err != nil {
		jsonResponse(c, ErrReadFileFailed, nil)
		return
	}
	jsonResponse(c, ErrSuccess, string(content))
}
