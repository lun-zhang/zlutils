package metric

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"testing"
	"time"
	"zlutils/code"
)

func TestMidRespCounterErr(t *testing.T) {
	const projectName = "zlutils"
	code.InitDefaultMetric(projectName)
	router := gin.New()
	router.Group(projectName).GET("metrics", Metrics)
	base := router.Group(projectName, MidRespCounterErr(code.RespIsServerErr,
		code.RespIsClientErr,
		code.DefaultServerErrorCounter,
		code.DefaultClientErrorCounter))
	{
		counterErr := base.Group("counter/err")
		counterErr.GET("server", func(c *gin.Context) {
			code.Send(c, nil, fmt.Errorf("s"))
		})
		counterErr.GET("client", func(c *gin.Context) {
			code.Send(c, nil, code.ClientErr.WithErrorf("c"))
		})
		counterErr.GET("no_ret", func(c *gin.Context) {
			c.JSON(http.StatusBadRequest, nil) //metric会记录为no_ret
		})
	}
	router.Run(":11116")
}

func TestMidRespCounterLantency(t *testing.T) {
	const projectName = "zlutils"
	InitDefaultMetric(projectName)
	router := gin.New()
	router.Group(projectName).GET("metrics", Metrics) //这样就不会把metric的调用记录下来
	base := router.Group(projectName, MidRespCounterLatency(DefaultRespCounter, DefaultRespLatency))
	base.GET("", func(c *gin.Context) {
		time.Sleep(time.Millisecond * 100)
		c.JSON(http.StatusOK, 1)
	})
	router.Run(":11117")
}
