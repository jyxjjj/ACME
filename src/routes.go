package main

import (
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

var r *gin.Engine
var addr, port string

func init() {
	addr = os.Getenv("ADDR")
	port = os.Getenv("PORT")
	if addr == "" {
		addr = "0.0.0.0"
	}
	if port == "" {
		port = "9504"
	}
}

func initRoutes() {
	r = gin.New()
	r.Use(Logger)
	r.Use(gin.Recovery())
	r.RemoveExtraSlash = true
	r.NoRoute(func(c *gin.Context) {
		c.String(404, "Not Found")
	})
	r.GET("/cert", authVerify, getCert)
	r.GET("/key", authVerify, getKey)
}

func Logger(c *gin.Context) {
	start := time.Now()
	c.Next()
	param := gin.LogFormatterParams{
		StatusCode:   c.Writer.Status(),
		Latency:      time.Since(start),
		ClientIP:     c.ClientIP(),
		Method:       c.Request.Method,
		Path:         c.Request.URL.Path,
		ErrorMessage: c.Errors.ByType(gin.ErrorTypePrivate).String(),
	}
	statusColor := param.StatusCodeColor()
	methodColor := param.MethodColor()
	resetColor := param.ResetColor()
	if param.Latency > time.Minute {
		param.Latency = param.Latency.Truncate(time.Second)
	}
	Log.Printf("[GIN] %s%3d%s|%13v|%15s|%s%-7s%s|%#v\n%s",
		statusColor, param.StatusCode, resetColor,
		param.Latency,
		param.ClientIP,
		methodColor, param.Method, resetColor,
		param.Path,
		param.ErrorMessage,
	)
}

func getCert(c *gin.Context) {
	if _, err := os.Stat(domainCrt); os.IsNotExist(err) {
		jsonResponse(c, ErrFileNotFound, nil)
		return
	}
	content, err := os.ReadFile(domainCrt)
	if err != nil {
		jsonResponse(c, ErrReadFileFailed, nil)
		return
	}
	jsonResponse(c, ErrSuccess, string(content))
}

func getKey(c *gin.Context) {
	if _, err := os.Stat(domainKey); os.IsNotExist(err) {
		jsonResponse(c, ErrFileNotFound, nil)
		return
	}
	content, err := os.ReadFile(domainKey)
	if err != nil {
		jsonResponse(c, ErrReadFileFailed, nil)
		return
	}
	jsonResponse(c, ErrSuccess, string(content))
}
