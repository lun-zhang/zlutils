package xray

import (
	"context"
	"fmt"
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
	router.GET("err/seg", code.Wrap(func(c *code.Context) {
		c.Send("seg err", f1(c.Request.Context()))
	}))
	endless.ListenAndServe(":11112", router)
}

func f1(ctx context.Context) (err error) {
	defer BeginSubsegment(&ctx)(&err)
	return f2(ctx)
}
func f2(ctx context.Context) (err error) {
	defer BeginSubsegment(&ctx)(&err)
	return fmt.Errorf("f2 err")
}
