package guard

import (
	"github.com/gin-gonic/gin"
	"testing"
	"xlbj-gitlab.xunlei.cn/oversea/zlutils/v6/code"
)

func TestMid(t *testing.T) {
	router := gin.New()
	router.Group("", Mid(nil)).GET("panic/default", p)
	router.Group("", Mid(code.Send)).GET("panic/code/default", p)
	router.Group("", Mid(func(c *gin.Context, data interface{}, err error) {
		code.Send(c, data, code.ServerErrPainc.WithError(err))
	})).GET("panic/code/with", p)
	router.Run(":11113")
}
func p(c *gin.Context) {
	panic("p")
}
