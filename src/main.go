package main

import (
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"os"
)

func main() {
	r := gin.Default()

	// default route
	r.NoRoute(func(c *gin.Context) {
		c.String(404, "Not Found")
	})

	r.GET("/cert", authVerify, getCert)
	r.GET("/key", authVerify, getKey)

	// run server
	addr := os.Getenv("ADDR")
	port := os.Getenv("PORT")
	r.Run(addr + ":" + port)
}

func getCert(c *gin.Context) {
	domain := c.Query("domain")
	if domain == "" {
		jsonResponse(c, ErrDomainRequired, nil)
		return
	}

	certPath := "./certs/" + domain + "/cert.crt"
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		jsonResponse(c, ErrFileNotFound, nil)
		return
	}
	content, err := ioutil.ReadFile(certPath)
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

	keyPath := "./certs/" + domain + "/priv.key"
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		jsonResponse(c, ErrFileNotFound, nil)
		return
	}
	content, err := ioutil.ReadFile(keyPath)
	if err != nil {
		jsonResponse(c, ErrReadFileFailed, nil)
		return
	}
	jsonResponse(c, ErrSuccess, string(content))
}
