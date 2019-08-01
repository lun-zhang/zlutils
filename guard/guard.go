package guard

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
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

type RecoverFunc func(errp *error)                         //把panic变成err传出
type BeforeFunc func(args ...interface{}) RecoverFunc      //这个虽然灵活，但是不便于在编译时发现错误
type BeforeCtxFunc func(ctxp *context.Context) RecoverFunc //会修改ctx

var DefaultBeforeCtx BeforeCtxFunc = func(ctxp *context.Context) RecoverFunc {
	return DefaultRecover
}

//默认的可以把panic转化成err，并打日志
var DefaultRecover RecoverFunc = func(errp *error) {
	if r := recover(); r != nil {
		err := fmt.Errorf("panic: %+v", r)
		logrus.WithError(err).Error()
		if errp != nil {
			*errp = err
		}
	}
}

var BeforeCtx = func(ctxp *context.Context) RecoverFunc {
	return DefaultBeforeCtx(ctxp)
}
