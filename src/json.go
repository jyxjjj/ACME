package main

import (
	"github.com/gin-gonic/gin"
)

func jsonResponse(c *gin.Context, errNo int, data interface{}) {
	err, ok := errorMap[errNo]
	if !ok {
		err = struct {
			code int
			msg  string
		}{-1, "Unknown error"}
	}
	if data == nil {
		data = []interface{}{}
	}
	c.JSON(200, gin.H{
		"code": err.code,
		"msg":  err.msg,
		"data": data,
	})
}
