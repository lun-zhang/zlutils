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
	router.Group("user/code", MidUser(code.SendClientErrHeader)).GET("", u)
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
	router.Group("operator/code", MidOperator(code.SendClientErrQuery)).GET("", o)
	router.Run(":11116")
}
