package xray

import (
	"context"
	"fmt"
	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/gin-gonic/gin"
	"testing"
	"zlutils/code"
	"zlutils/guard"
	_ "zlutils/logger"
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
	router.GET("replace", func(c *gin.Context) {
		s := &S{}
		s.f(c.Request.Context())
		code.Send(c, "r", nil)
	})
	router.Run(":11112")
}

type S struct {
}

func (s *S) f(ctx context.Context) (err error) {
	defer guard.BeforeCtx(&ctx)(&err)
	return
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

func TestGetTraceId(t *testing.T) {
	ctx, seg := xray.BeginSegment(context.Background(), "test")
	if err := mustGetTraceId(ctx, 1, seg.TraceID); err != nil {
		t.Fatal(err)
	}
}

func mustGetTraceId(ctx context.Context, dep int, initTraceId string) (err error) {
	defer guard.BeforeCtx(&ctx)(&err)
	if dep > 100 {
		return nil
	}

	traceId := GetTraceId(ctx)
	if traceId == "" {
		return fmt.Errorf("trace id empty")
	}
	if traceId != initTraceId {
		return fmt.Errorf("trace id diff:%s %s", traceId, initTraceId)
	}

	return mustGetTraceId(ctx, dep+1, initTraceId)
}

func TestPanic(t *testing.T) {
	guard.DoBeforeCtx = DoBeforeCtx
	guard.DoAfter = DoAfter
	ctx, _ := xray.BeginSegment(context.TODO(), "test")
	p(ctx)
}

func p(ctx context.Context) (err error) {
	defer guard.BeforeCtx(&ctx)(&err)

	panic(1)
}
