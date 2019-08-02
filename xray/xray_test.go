package xray

import (
	"context"
	"fmt"
	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/gin-gonic/gin"
	"testing"
	"zlutils/code"
	"zlutils/guard"
)

func TestMid(t *testing.T) {
	guard.DoBeforeCtx = DoBeforeCtx
	guard.DoAfter = DoAfter
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
	router.Run(":11112")
}

func f1(ctx context.Context) (err error) {
	fmt.Println(xray.GetSegment(ctx))
	defer guard.BeforeCtx(&ctx)(&err)
	return f2(ctx)
}
func f2(ctx context.Context) (err error) {
	defer guard.BeforeCtx(&ctx)(&err)
	return nil
}

func TestBeginSeg(t *testing.T) {
	guard.DoBeforeCtx = DoBeforeCtx
	guard.DoAfter = DoAfter
	ctx := context.Background()
	fmt.Println(f1(ctx))
}
