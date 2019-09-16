package guard

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	"time"
	"zlutils/caller"
)

func Mid(sendServerErrPanic func(c *gin.Context, data interface{}, err error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				if sendServerErrPanic != nil {
					sendServerErrPanic(c, nil, fmt.Errorf("panic: %+v", rec)) //用户自定义处理
				} else {
					c.JSON(http.StatusInternalServerError, nil) //默认返回500
				}
			}
		}()
		c.Next()
	}
}

type (
	AfterFunc       func(errp *error)                     //会修改err,把panic变成err传出
	BeforeFunc      func(args ...interface{}) AfterFunc   //这个虽然灵活，但是不便于在编译时发现错误
	BeforeCtxFunc   func(ctxp *context.Context) AfterFunc //会修改ctx
	DoBeforeCtxFunc func(ctxp *context.Context) (args []interface{})
	DoAfterFunc     func(err error, args ...interface{})
)

var (
	//NOTE: 允许用户函数开始前执行自定义方法，返回自定义的值
	DoBeforeCtx DoBeforeCtxFunc = func(ctxp *context.Context) (args []interface{}) { return }
	//NOTE: 允许用户在函数结束时候执行自定义方法，第一个参数是本身的err，或panic的err(要不要换成errp?)
	DoAfter DoAfterFunc = func(err error, args ...interface{}) {}
)

//NOTE: 如果不传，则用默认的（似乎不需要这个）
//func InitDoCtxFunc(doBeforeCtx DoBeforeCtxFunc, doAfter DoAfterFunc) {
//	if doBeforeCtx != nil {
//		DoBeforeCtx = doBeforeCtx
//	}
//	if doAfter != nil {
//		DoAfter = doAfter
//	}
//}

var PanicToError = func(i interface{}) error {
	return fmt.Errorf("panic: %+v", i)
}

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
				err = PanicToError(r)
				logrus.WithError(err).Error() //一般不在其他地方打
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
