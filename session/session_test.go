package session

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"testing"
	"zlutils/code"
)

func TestMidUser(t *testing.T) {
	router := gin.New()
	router.Group("user/default", MidUser(nil)).GET("", u)

	router.Group("user/code/default", MidUser(code.Send)).GET("", u) //NOTE: 未定义的错误会被认为是服务器错误
	router.Group("user/code/with", MidUser(func(c *gin.Context, data interface{}, err error) {
		code.Send(c, data, code.ClientErrHeader.WithError(err)) //NOTE: 因此应当修正为客户端错误
	})).GET("", u)

	router.Run(":11115")
}
func u(c *gin.Context) {
	c.JSON(http.StatusOK, GetUser(c))
}
func o(c *gin.Context) {
	c.JSON(http.StatusOK, GetOperator(c))
}

func TestMidOperator(t *testing.T) {
	router := gin.New()
	router.Group("operator/default", MidOperator(nil)).GET("", o)

	router.Group("operator/code/default", MidOperator(code.Send)).GET("", o) //NOTE: 未定义的错误会被认为是服务器错误
	router.Group("operator/code/with", MidOperator(func(c *gin.Context, data interface{}, err error) {
		code.Send(c, data, code.ClientErrQuery.WithError(err)) //NOTE: 因此应当修正为客户端错误
	})).GET("", o)

	router.Run(":11116")
}
