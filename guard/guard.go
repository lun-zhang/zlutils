package guard

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"time"
	"zlutils/caller"
	"zlutils/code"
	"zlutils/logger"
)

func Mid() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				//FIXME: 依赖了code包，作为更基础的包似乎应该解依赖
				err := code.ServerErrPainc.WithErrorf("panic: %+v", rec)
				logrus.WithContext(c.Request.Context()).WithError(err).WithField(logger.FieldStack, caller.DebugStack()).Error()
				code.Send(c, nil, err)
			}
		}()
		c.Next()
	}
}

type (
	AfterFunc func(errp *error) //会修改err,把panic变成err传出
	//BeforeFunc      func(args ...interface{}) AfterFunc   //这个虽然灵活，但是不便于在编译时发现错误
	//BeforeCtxFunc   func(ctxp *context.Context) AfterFunc //会修改ctx
	DoBeforeCtxFunc func(ctxp *context.Context) (args []interface{})
	DoAfterFunc     func(err error, args ...interface{})
)

var (
	//允许用户函数开始前执行自定义方法，返回自定义的值
	DoBeforeCtx DoBeforeCtxFunc = func(ctxp *context.Context) (args []interface{}) { return }
	//允许用户在函数结束时候执行自定义方法，第一个参数是本身的err，或panic的err(要不要换成errp?)
	DoAfter DoAfterFunc = func(err error, args ...interface{}) {}
)

func BeforeCtx(ctxp *context.Context) AfterFunc {
	start := time.Now()
	name := caller.Caller(2)
	args := DoBeforeCtx(ctxp)
	if MetricUnderway != nil {
		MetricUnderway(name).Inc()
	}
	return func() AfterFunc {
		return func(errp *error) {
			var err error
			if r := recover(); r != nil {
				err = code.ServerErrPainc.WithErrorf("panic: %+v", r)
				logrus.WithContext(*ctxp).WithError(err).Error() //一般不在其他地方打
				if errp != nil {
					*errp = err
				}
			} else {
				if errp != nil {
					err = *errp
				}
			}
			if MetricCounter != nil {
				MetricCounter(name).Inc()
			}
			if MetricLatency != nil {
				MetricLatency(name).Observe(time.Now().Sub(start).Seconds() * 1000)
			}
			if MetricUnderway != nil {
				MetricUnderway(name).Dec()
			}
			DoAfter(err, args...) //即使errp==nil，也把panic的err传入
		}
	}()
}
