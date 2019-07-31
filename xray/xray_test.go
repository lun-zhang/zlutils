package xray

import (
	"github.com/fvbock/endless"
	"github.com/gin-gonic/gin"
	"testing"
	"zlutils/code"
)

func TestMid(t *testing.T) {
	router := gin.New()
	router.Use(Mid("zlutils", nil,
		code.RespIsServerErr, code.RespIsClientErr))
	router.GET("ok", code.Wrap(func(c *code.Context) {
		c.Send("ok", nil)
	}))
	router.GET("err/server", code.Wrap(func(c *code.Context) {
		c.Send("server err", code.ServerErr)
	}))
	router.GET("err/client", code.Wrap(func(c *code.Context) {
		c.Send("client err", code.ClientErr)
	}))
	endless.ListenAndServe(":11112", router)
}
