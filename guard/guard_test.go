package guard

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"testing"
	"time"
	"zlutils/code"
	"zlutils/metric"
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

func TestMetric(t *testing.T) {
	const projectName = "zlutils"
	InitDefaultMetric(projectName)
	router := gin.New()
	router.Group(projectName).GET("metrics", metric.Metrics)
	go func() {
		for {
			f3()
			time.Sleep(time.Second)
		}
	}()
	router.Run(":11120")
}

func f3() {
	defer BeforeCtx(nil)(nil)
	time.Sleep(time.Second)
}
