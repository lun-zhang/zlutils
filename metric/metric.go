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
)

var GetEndpoint = func(c *gin.Context) string {
	return fmt.Sprintf("%s-%s", c.Request.URL.Path, c.Request.Method)
}

func MidRespCounterLatency(projectName string) gin.HandlerFunc {
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

	prometheus.MustRegister(
		defaultRespCounter,
		defaultRespLatency,
	)

	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latency := time.Now().Sub(start)
		defaultRespLatency.WithLabelValues(GetEndpoint(c)).Observe(latency.Seconds() * 1000)
		defaultRespCounter.WithLabelValues(GetEndpoint(c)).Inc()
	}
}

func Metrics(c *gin.Context) {
	handler := promhttp.Handler()
	handler.ServeHTTP(c.Writer, c.Request)
}
