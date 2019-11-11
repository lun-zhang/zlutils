package code

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"testing"
	"zlutils/metric"
)

const projectName = "zlutils"

func TestMidRespCounterErr(t *testing.T) {
	router := gin.New()
	router.Group(projectName).GET("metrics", metric.Metrics)
	base := router.Group(projectName, MidRespCounterErr(projectName))
	{
		counterErr := base.Group("counter/err")
		counterErr.GET("server", func(c *gin.Context) {
			Send(c, nil, fmt.Errorf("s"))
		})
		counterErr.GET("client", func(c *gin.Context) {
			Send(c, nil, ClientErr.WithErrorf("c"))
		})
		counterErr.GET("no_ret", func(c *gin.Context) {
			c.JSON(http.StatusBadRequest, nil) //metric会记录为no_ret
		})
	}
	router.Run(":11116")
}
