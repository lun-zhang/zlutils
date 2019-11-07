package metric

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
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
		code.ServerErrorCounter,
		code.ClientErrorCounter))
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

const projectName = "zlutils"

func TestMidRespCounterLantency(t *testing.T) {

	InitDefaultMetric(projectName)
	router := gin.New()
	router.Group(projectName).GET("metrics", Metrics) //这样就不会把metric的调用记录下来
	base := router.Group(projectName, MidRespCounterLatency())
	base.GET("", func(c *gin.Context) {
		time.Sleep(time.Millisecond * 100)
		c.JSON(http.StatusOK, 1)
	})
	router.Run(":11117")
}

//避免path参数不同而metric变成不同的线
func TestMidPath(t *testing.T) {
	GetEndpoint = func(c *gin.Context) string {
		path := c.Request.URL.Path
		if pre := "/zlutils/path/"; strings.HasPrefix(path, pre) {
			path = pre + ":id"
		}
		return fmt.Sprintf("%s-%s", path, c.Request.Method)
	}
	InitDefaultMetric(projectName)
	router := gin.New()
	router.Group(projectName).GET("metrics", Metrics) //这样就不会把metric的调用记录下来
	base := router.Group(projectName, MidRespCounterLatency())
	base.GET("path/:id", func(c *gin.Context) {
		time.Sleep(time.Millisecond * 100)
		c.JSON(http.StatusOK, c.Request.URL.Path)
	})
	router.Run(":11118")
}
