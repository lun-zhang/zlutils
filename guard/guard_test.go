package guard

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"testing"
	"zlutils/code"
)

func TestMid(t *testing.T) {
	router := gin.New()
	router.Group("", Mid(nil)).GET("panic/default", p)
	router.Group("", Mid(code.SendServerErrPanic)).GET("panic/code", p)
	router.Run(":11113")
}
func p(c *gin.Context) {
	panic("p")
}

func TestBefore(t *testing.T) {
	fmt.Println(f1())
}

func f1() (err error) {
	defer BeforeCtx(nil)(&err)
	return f2()
}
func f2() (err error) {
	defer BeforeCtx(nil)(&err)
	panic(1)
}
