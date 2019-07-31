package metric

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"time"
)

var (
	historyBuckets = []float64{10., 20., 30., 50., 80., 100., 200., 300., 500., 1000., 2000., 3000.}

	ResponseCounter *prometheus.CounterVec   //请求次数
	ResponseLatency *prometheus.HistogramVec //请求耗时，用于alert

	ServerErrorCounter *prometheus.CounterVec //服务器错误，用于alter
	ClientErrorCounter *prometheus.CounterVec //客户端错误

	MysqlCounter *prometheus.CounterVec   //mysql查询次数
	MysqlLatency *prometheus.HistogramVec //mysql耗时

	FuncCounter *prometheus.CounterVec   //func次数
	FuncLatency *prometheus.HistogramVec //func耗时，虽然xray里也有

	LogCounter *prometheus.CounterVec //log次数
	//写日志很快所以没有计时
)

func Init(projectName string) {
	ResponseCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: fmt.Sprintf("%s_requests_total", projectName),
			Help: "Total request counts",
		},
		[]string{"endpoint"},
	)
	ServerErrorCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: fmt.Sprintf("%s_server_error_total", projectName),
			Help: "Total Server Error counts",
		},
		[]string{"endpoint", "ret"},
	)
	ClientErrorCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: fmt.Sprintf("%s_client_error_total", projectName),
			Help: "Total Client Error counts",
		},
		[]string{"endpoint", "ret"},
	)
	ResponseLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    fmt.Sprintf("%s_response_latency_millisecond", projectName),
			Help:    "Response latency (millisecond)",
			Buckets: historyBuckets,
		},
		[]string{"endpoint"},
	)
	MysqlCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: fmt.Sprintf("%s_mysql_total", projectName),
			Help: "Total Mysql counts",
		},
		[]string{"query"},
	)
	MysqlLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    fmt.Sprintf("%s_mysql_latency_millisecond", projectName),
			Help:    "Mysql latency (millisecond)",
			Buckets: historyBuckets,
		},
		[]string{"query"},
	)

	FuncCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: fmt.Sprintf("%s_func_total", projectName),
			Help: "Total Func counts",
		},
		[]string{"func"},
	)
	FuncLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    fmt.Sprintf("%s_func_latency_millisecond", projectName),
			Help:    "Func latency (millisecond)",
			Buckets: historyBuckets,
		},
		[]string{"func"},
	)

	LogCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: fmt.Sprintf("%s_log_total", projectName),
			Help: "Total Log counts",
		},
		[]string{"level"},
	)

	prometheus.MustRegister(
		ResponseCounter,
		ServerErrorCounter,
		ClientErrorCounter,
		ResponseLatency,
		MysqlCounter,
		MysqlLatency,
		FuncCounter,
		FuncLatency,
		LogCounter,
	)
}

func MidCounter(isServerErr, isClientErr fcb,
	responseCounter, serverErrCounter, clientErrCounter fcc) gin.HandlerFunc {
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
		if responseCounter != nil {
			responseCounter(c).Inc()
		}
	}
}

type fcb func(*gin.Context) bool
type fcc func(*gin.Context) prometheus.Counter
type fco func(*gin.Context) prometheus.Observer

//不需要skip，不需要的接口不用此中间件即可
func MidLantency(responseLatency fco) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next() //这里面可能发生panic
		latency := time.Now().Sub(start)
		if responseLatency != nil {
			responseLatency(c).Observe(latency.Seconds() * 1000)
		}
	}
}

func Serve(c *gin.Context) {
	handler := promhttp.Handler()
	handler.ServeHTTP(c.Writer, c.Request)
}
