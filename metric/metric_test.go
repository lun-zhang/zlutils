package metric

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
	"testing"
	"time"
)

const projectName = "zlutils"

func TestMidRespCounterLantency(t *testing.T) {
	router := gin.New()
	router.Group(projectName).GET("metrics", Metrics) //这样就不会把metric的调用记录下来
	base := router.Group(projectName, MidRespCounterLatency(projectName))
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
	router := gin.New()
	router.Group(projectName).GET("metrics", Metrics) //这样就不会把metric的调用记录下来
	base := router.Group(projectName, MidRespCounterLatency(projectName))
	base.GET("path/:id", func(c *gin.Context) {
		time.Sleep(time.Millisecond * 100)
		c.JSON(http.StatusOK, c.Request.URL.Path)
	})
	router.Run(":11118")
}
