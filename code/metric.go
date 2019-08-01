package code

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"strconv"
)

var (
	defaultServerErrorCounter *prometheus.CounterVec //服务器错误，用于alter
	defaultClientErrorCounter *prometheus.CounterVec //客户端错误
)

func InitDefaultMetric(projectName string) {
	defaultServerErrorCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: fmt.Sprintf("%s_server_error_total", projectName),
			Help: "Total Server Error counts",
		},
		[]string{"endpoint", "ret"},
	)
	defaultClientErrorCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: fmt.Sprintf("%s_client_error_total", projectName),
			Help: "Total Client Error counts",
		},
		[]string{"endpoint", "ret"},
	)
	prometheus.MustRegister(
		defaultServerErrorCounter,
		defaultClientErrorCounter,
	)
}

func getEndpoint(c *gin.Context) string {
	return fmt.Sprintf("%s-%s", c.Request.URL.Path, c.Request.Method)
}

func getRet(c *gin.Context) string {
	ret, ok := GetRet(c)
	if !ok {
		return "no_ret"
	}
	return strconv.Itoa(ret)
}

func DefaultServerErrorCounter(c *gin.Context) prometheus.Counter {
	return defaultServerErrorCounter.WithLabelValues(getEndpoint(c), getRet(c))
}
func DefaultClientErrorCounter(c *gin.Context) prometheus.Counter {
	return defaultClientErrorCounter.WithLabelValues(getEndpoint(c), getRet(c))
}
