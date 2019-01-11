package zlutils

import (
	"bytes"
	"context"
	"fmt"
	"github.com/aws/aws-xray-sdk-go/header"
	"github.com/aws/aws-xray-sdk-go/xray"
	xlog "github.com/cihub/seelog"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"net/http"
	"runtime"
	"strconv"
	"time"
)

var (
	ProjectName string

	historyBuckets = [...]float64{
		10., 20., 30., 50., 80., 100., 200., 300., 500., 1000., 2000., 3000.}

	ResponseCounter *prometheus.CounterVec
	ErrorCounter    *prometheus.CounterVec
	ResponseLatency *prometheus.HistogramVec

	sn *xray.FixedSegmentNamer
)

func InitTrace() {
	ResponseCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: fmt.Sprintf("%s_requests_total", ProjectName),
			Help: "Total request counts",
		},
		[]string{"method", "endpoint"},
	)
	ErrorCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: fmt.Sprintf("%s_error_total", ProjectName),
			Help: "Total Error counts",
		},
		[]string{"method", "endpoint"},
	)
	ResponseLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    fmt.Sprintf("%s_response_latency_millisecond", ProjectName),
			Help:    "Response latency (millisecond)",
			Buckets: historyBuckets[:],
		},
		[]string{"method", "endpoint"},
	)

	sn = xray.NewFixedSegmentNamer(ProjectName)

	prometheus.MustRegister(ResponseCounter)
	prometheus.MustRegister(ErrorCounter)
	prometheus.MustRegister(ResponseLatency)

	xray.Configure(xray.Config{
		DaemonAddr:     "127.0.0.1:3000",
		LogLevel:       "info",
		ServiceVersion: "1.0.0",
	})
}

func Metrics(notLogged ...string) gin.HandlerFunc {
	var skip map[string]struct{}

	if length := len(notLogged); length > 0 {
		skip = make(map[string]struct{}, length)

		for _, path := range notLogged {
			skip[path] = struct{}{}
		}
	}

	return func(c *gin.Context) {
		r := c.Request
		path := r.URL.Path
		if _, ok := skip[path]; ok {
			// Process request
			c.Next()
		} else {
			// Start timer
			start := time.Now()

			// xray http trace before operation
			name := sn.Name(c.Request.Host)
			traceHeader := header.FromString(r.Header.Get("x-amzn-trace-id"))
			ctx, seg := xray.NewSegmentFromHeader(r.Context(), name, traceHeader)
			r = r.WithContext(ctx)
			c.Request = r
			seg.Lock()

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
				xlog.Tracef("SamplingStrategy decided: %t", seg.Sampled)
			}
			if traceHeader.SamplingDecision == header.Requested {
				respHeader.WriteString(";Sampled=")
				respHeader.WriteString(strconv.Itoa(btoi(seg.Sampled)))
			}

			c.Writer.Header().Set("x-amzn-trace-id", respHeader.String())
			seg.Unlock()

			// Process request
			c.Next()

			clientIP := c.ClientIP()
			method := c.Request.Method
			statusCode := c.Writer.Status()
			comment := c.Errors.ByType(gin.ErrorTypePrivate).String()

			seg.Lock()
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
			seg.Unlock()
			seg.Close(nil)

			// Stop timer
			end := time.Now()
			latency := end.Sub(start)

			logrus.WithFields(logrus.Fields{
				"statusCode": statusCode,
				"latency":    fmt.Sprintf("%v", latency),
				"clientIP":   clientIP,
				"method":     method,
				"path":       path,
				"comment":    comment,
			}).Info()

			if statusCode != http.StatusNotFound {
				elapsed := latency.Seconds() * 1000.0
				ResponseCounter.WithLabelValues(method, path).Inc()
				ErrorCounter.WithLabelValues(strconv.FormatInt(int64(statusCode), 10),
					fmt.Sprintf("%s-%s", path, method)).Inc()
				ResponseLatency.WithLabelValues(method, path).Observe(elapsed)
			}
		}

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
		if len(s) < len(ProjectName) || s[:len(ProjectName)] != ProjectName {
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
func BeginSubsegment(ctxp *context.Context) (seg *xray.Segment) {
	name := GetSource(2)
	*ctxp, seg = xray.BeginSubsegment(*ctxp, name)
	return
}
func BeginSegment(ctxp *context.Context) (seg *xray.Segment) {
	name := GetSource(2)
	*ctxp, seg = xray.BeginSegment(*ctxp, name)
	return
}
