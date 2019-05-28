package zlutils

import (
	"bytes"
	"context"
	"fmt"
	"github.com/aws/aws-xray-sdk-go/header"
	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"net/http"
	"regexp"
	"runtime"
	"strconv"
	"time"
)

var (
	ProjectName string

	historyBuckets = [...]float64{
		10., 20., 30., 50., 80., 100., 200., 300., 500., 1000., 2000., 3000.}

	ResponseCounter *prometheus.CounterVec   //请求次数
	ResponseLatency *prometheus.HistogramVec //请求耗时，用于alert

	ServerErrorCounter *prometheus.CounterVec //服务器错误，用于alter
	ClientErrorCounter *prometheus.CounterVec //客户端错误

	MysqlCounter *prometheus.CounterVec   //mysql查询次数
	MysqlLatency *prometheus.HistogramVec //mysql耗时

	LogCounter *prometheus.CounterVec //log次数

	sn *xray.FixedSegmentNamer
)

var pr *regexp.Regexp

func InitTrace() {
	pr = regexp.MustCompile(ProjectName)
	ResponseCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: fmt.Sprintf("%s_requests_total", ProjectName),
			Help: "Total request counts",
		},
		[]string{"endpoint"},
	)
	ServerErrorCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: fmt.Sprintf("%s_server_error_total", ProjectName),
			Help: "Total Server Error counts",
		},
		[]string{"endpoint", "ret"},
	)
	ClientErrorCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: fmt.Sprintf("%s_client_error_total", ProjectName),
			Help: "Total Client Error counts",
		},
		[]string{"endpoint", "ret"},
	)
	ResponseLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    fmt.Sprintf("%s_response_latency_millisecond", ProjectName),
			Help:    "Response latency (millisecond)",
			Buckets: historyBuckets[:],
		},
		[]string{"endpoint"},
	)
	MysqlCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: fmt.Sprintf("%s_mysql_total", ProjectName),
			Help: "Total Mysql counts",
		},
		[]string{"query"},
	)
	MysqlLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    fmt.Sprintf("%s_mysql_latency_millisecond", ProjectName),
			Help:    "Mysql latency (millisecond)",
			Buckets: historyBuckets[:],
		},
		[]string{"query"},
	)

	LogCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: fmt.Sprintf("%s_log_total", ProjectName),
			Help: "Total Log counts",
		},
		[]string{"level"},
	)

	sn = xray.NewFixedSegmentNamer(ProjectName)

	prometheus.MustRegister(
		ResponseCounter,
		ServerErrorCounter,
		ClientErrorCounter,
		ResponseLatency,
		MysqlCounter,
		MysqlLatency,
		LogCounter,
	)

	xray.Configure(xray.Config{
		DaemonAddr:     "127.0.0.1:3000",
		LogLevel:       "info",
		ServiceVersion: "1.0.0",
	})
}

//不需要skip，不需要的接口不用此中间件即可
func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		r := c.Request
		// Start timer
		start := time.Now()

		// xray http trace before operation
		name := sn.Name(c.Request.Host)
		traceHeader := header.FromString(r.Header.Get("x-amzn-trace-id"))
		ctx, seg := xray.NewSegmentFromHeader(r.Context(), name, traceHeader)
		r = r.WithContext(ctx)
		c.Request = r
		//seg.Lock()

		scheme := "https://"
		if r.TLS == nil {
			scheme = "http://"
		}
		seg.GetHTTP().GetRequest().Method = r.Method
		seg.GetHTTP().GetRequest().URL = scheme + r.Host + r.URL.Path
		seg.GetHTTP().GetRequest().ClientIP, seg.GetHTTP().GetRequest().XForwardedFor = clientIP(r)
		seg.GetHTTP().GetRequest().UserAgent = r.UserAgent()

		var respHeader bytes.Buffer
		respHeader.WriteString("Root=")
		respHeader.WriteString(seg.TraceID)

		if traceHeader.SamplingDecision != header.Sampled && traceHeader.SamplingDecision != header.NotSampled {
			seg.Sampled = seg.ParentSegment.GetConfiguration().SamplingStrategy.ShouldTrace(r.Host, r.URL.Path, r.Method)
			logrus.Tracef("SamplingStrategy decided: %t", seg.Sampled)
		}
		if traceHeader.SamplingDecision == header.Requested {
			respHeader.WriteString(";Sampled=")
			respHeader.WriteString(strconv.Itoa(btoi(seg.Sampled)))
		}

		c.Writer.Header().Set("x-amzn-trace-id", respHeader.String())
		//seg.Unlock()

		defer func() {
			rec := recover()
			if rec != nil {
				CodeSend(c, nil, CodeServerMidPaincErr.WithErrorf("panic recover: %s", rec))
			}

			clientIP := c.ClientIP()
			statusCode := c.Writer.Status()
			comment := c.Errors.ByType(gin.ErrorTypePrivate).String()

			//seg.Lock()
			seg.GetHTTP().GetResponse().Status = c.Writer.Status()
			seg.GetHTTP().GetResponse().ContentLength, _ = strconv.Atoi(c.Writer.Header().Get("Content-Length"))

			if statusCode >= 400 && statusCode < 500 {
				seg.Error = true
			}
			if statusCode == 429 {
				seg.Throttle = true
			}
			if statusCode >= 500 && statusCode < 600 {
				seg.Fault = true
			}

			// Stop timer
			latency := time.Now().Sub(start)

			if statusCode == http.StatusNotFound {
				c.Set(KeyRet, CodeClient404Err.Ret) //404也打日志
			}
			endpoint := fmt.Sprintf("%s-%s", r.URL.Path, c.Request.Method)
			entry := logrus.WithFields(logrus.Fields{
				"statusCode": statusCode,
				"latency":    latency.String(),
				"clientIP":   clientIP,
				"endpoint":   endpoint,
				"comment":    comment,
			})

			ret, ok := c.Value(KeyRet).(int) //NOTE: 如果返回没调CodeSend就没有ret，避免panic
			if ok {
				if ret >= 4000 && ret < 5000 {
					ClientErrorCounter.WithLabelValues(endpoint, strconv.Itoa(ret)).Inc()
				} else if ret >= 5000 && ret < 6000 {
					ServerErrorCounter.WithLabelValues(endpoint, strconv.Itoa(ret)).Inc()
				}
				entry = entry.WithField("ret", ret)
			} else {
				entry.Warnf("invalid ret:%+v", c.Value(KeyRet))
			}

			ResponseCounter.WithLabelValues(endpoint).Inc()
			ResponseLatency.WithLabelValues(endpoint).Observe(latency.Seconds() * 1000.0)

			entry.Info()
			if latency > 500*time.Millisecond {
				entry.Warn("slow api")
			}
			if rec != nil {
				entry.WithFields(logrus.Fields{
					"stack":   GetStack(3),
					"recover": rec,
				}).Error()
			}
			//seg.Unlock()
			seg.Close(nil)
		}()
		// Process request
		c.Next() //这里面可能发生panic
	}
}

func GetMetrics(c *gin.Context) {
	handler := promhttp.Handler()
	handler.ServeHTTP(c.Writer, c.Request)
}

const unknown = "unknown"

func GetStack(skip int) (names []string) {
	for i := skip; ; i++ {
		s := GetSource(i)
		if s == unknown {
			break
		}
		if !pr.MatchString(s) {
			continue
		}
		names = append(names, s)
	}
	return
}

func GetSource(skip int) (name string) {
	name = unknown
	if pc, _, line, ok := runtime.Caller(skip); ok {
		name = fmt.Sprintf("%s:%d", runtime.FuncForPC(pc).Name(), line)
	}
	return
}

//凡是调用了该函数，再去调用其他函数时，传递的都是sub ctx
//没必要记录err，因为err携带信息太少，看日志才行，而且同一个err没必要每个函数都记录一次
func BeginSubsegment(ctxp *context.Context) func() {
	var seg *xray.Segment
	*ctxp, seg = xray.BeginSubsegment(*ctxp, GetSource(2))
	return func() { CloseSeg(seg) }
}
func BeginSegment(ctxp *context.Context) func() {
	var seg *xray.Segment
	*ctxp, seg = xray.BeginSegment(*ctxp, GetSource(2))
	return func() { CloseSeg(seg) }
}

func CloseSeg(seg *xray.Segment) {
	r := recover() //NOTE: 即使panic也要close
	seg.Close(nil)
	if r != nil {
		panic(r) //NOTE： close后把panic抛出
	}
}
