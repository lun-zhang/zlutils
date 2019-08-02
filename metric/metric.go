package metric

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"time"
)

var (
	HistoryBuckets = []float64{10., 20., 30., 50., 80., 100., 200., 300., 500., 1000., 2000., 3000.}

	defaultFuncCounter *prometheus.CounterVec   //func次数
	defaultFuncLatency *prometheus.HistogramVec //func耗时，虽然xray里也有
)

func InitDefaultMetric(projectName string) {
	//请求次数
	defaultRespCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: fmt.Sprintf("%s_requests_total", projectName),
			Help: "Total request counts",
		},
		[]string{"endpoint"},
	)
	//请求耗时，用于alert
	defaultRespLatency := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    fmt.Sprintf("%s_response_latency_millisecond", projectName),
			Help:    "Response latency (millisecond)",
			Buckets: HistoryBuckets,
		},
		[]string{"endpoint"},
	)

	defaultFuncCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: fmt.Sprintf("%s_func_total", projectName),
			Help: "Total Func counts",
		},
		[]string{"func"},
	)
	defaultFuncLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    fmt.Sprintf("%s_func_latency_millisecond", projectName),
			Help:    "Func latency (millisecond)",
			Buckets: HistoryBuckets,
		},
		[]string{"func"},
	)

	prometheus.MustRegister(
		defaultRespCounter,
		defaultRespLatency,
		defaultFuncCounter,
		defaultFuncLatency,
	)

	RespCounter = func(c *gin.Context) prometheus.Counter {
		return defaultRespCounter.WithLabelValues(getEndpoint(c))
	}

	RespLatency = func(c *gin.Context) prometheus.Observer {
		return defaultRespLatency.WithLabelValues(getEndpoint(c))
	}
}

func getEndpoint(c *gin.Context) string {
	return fmt.Sprintf("%s-%s", c.Request.URL.Path, c.Request.Method)
}

var (
	RespCounter fcc
	RespLatency fco
)

func MidRespCounterErr(isServerErr, isClientErr fcb,
	serverErrCounter, clientErrCounter fcc) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		statusCode := c.Writer.Status()
		if statusCode >= 400 && statusCode < 500 ||
			isClientErr != nil && isClientErr(c) {
			if clientErrCounter != nil {
				clientErrCounter(c).Inc()
			}
		}
		if statusCode >= 500 && statusCode < 600 ||
			isServerErr != nil && isServerErr(c) {
			if serverErrCounter != nil {
				serverErrCounter(c).Inc()
			}
		}
	}
}

type fcb func(*gin.Context) bool
type fcc func(*gin.Context) prometheus.Counter
type fco func(*gin.Context) prometheus.Observer

func MidRespCounterLatency() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latency := time.Now().Sub(start)
		if RespLatency != nil {
			RespLatency(c).Observe(latency.Seconds() * 1000)
		}
		if RespCounter != nil {
			RespCounter(c).Inc()
		}
	}
}

func Metrics(c *gin.Context) {
	handler := promhttp.Handler()
	handler.ServeHTTP(c.Writer, c.Request)
}
