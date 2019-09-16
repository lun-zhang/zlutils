package code

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"net/http"
	"strconv"
)

func InitDefaultMetric(projectName string) {
	serverErrorCounter := prometheus.NewCounterVec( //服务器错误，用于alert
		prometheus.CounterOpts{
			Name: fmt.Sprintf("%s_server_error_total", projectName),
			Help: "Total Server Error counts",
		},
		[]string{"endpoint", "ret"},
	)
	clientErrorCounter := prometheus.NewCounterVec( //客户端错误
		prometheus.CounterOpts{
			Name: fmt.Sprintf("%s_client_error_total", projectName),
			Help: "Total Client Error counts",
		},
		[]string{"endpoint", "ret"},
	)
	prometheus.MustRegister(
		serverErrorCounter,
		clientErrorCounter,
	)

	ServerErrorCounter = func(c *gin.Context) prometheus.Counter {
		return serverErrorCounter.WithLabelValues(getEndpoint(c), getRet(c))
	}
	ClientErrorCounter = func(c *gin.Context) prometheus.Counter {
		return clientErrorCounter.WithLabelValues(getEndpoint(c), getRet(c))
	}
}

func getEndpoint(c *gin.Context) string {
	return fmt.Sprintf("%s-%s", c.Request.URL.Path, c.Request.Method)
}

func getRet(c *gin.Context) string {
	ret, ok := GetRet(c)
	if !ok {
		if c.Writer.Status() == http.StatusNotFound {
			ret = ClientErr404.Ret
		} else {
			return "no_ret"
		}
	}
	return strconv.Itoa(ret)
}

var (
	ServerErrorCounter func(*gin.Context) prometheus.Counter
	ClientErrorCounter func(*gin.Context) prometheus.Counter
)
