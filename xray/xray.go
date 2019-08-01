package xray

import (
	"bytes"
	"context"
	"fmt"
	"github.com/aws/aws-xray-sdk-go/header"
	"github.com/aws/aws-xray-sdk-go/strategy/sampling"
	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net"
	"net/http"
	"strconv"
	"strings"
	"zlutils/caller"
	"zlutils/guard"
)

//用于填充xray的中间件
func Mid(projectName string, sample []byte,
	isServerErr, isClientErr func(*gin.Context) bool) gin.HandlerFunc {
	if sample == nil {
		sample = []byte(`{
  "version": 1,
  "default": {
    "fixed_target": 1,
    "rate": 0.05
  }
}`)
	}
	ss, err := sampling.NewLocalizedStrategyFromJSONBytes(sample)
	if err != nil {
		logrus.WithError(err).Fatal()
	}
	if err = xray.Configure(xray.Config{
		DaemonAddr:       "127.0.0.1:3000",
		LogLevel:         "info",
		ServiceVersion:   "1.0.0",
		SamplingStrategy: ss,
	}); err != nil {
		logrus.WithError(err).Fatal()
	}
	sn := xray.NewFixedSegmentNamer(projectName)

	return func(c *gin.Context) {
		r := c.Request
		// Start timer
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

		c.Next() //NOTE:这里不可抛出panic，必须在次之前被捕获

		statusCode := c.Writer.Status()

		//seg.Lock()
		seg.GetHTTP().GetResponse().Status = c.Writer.Status()
		seg.GetHTTP().GetResponse().ContentLength, _ = strconv.Atoi(c.Writer.Header().Get("Content-Length"))

		if statusCode >= 400 && statusCode < 500 ||
			isClientErr != nil && isClientErr(c) {
			seg.Error = true
		}
		if statusCode == 429 {
			seg.Throttle = true
		}
		if statusCode >= 500 && statusCode < 600 ||
			isServerErr != nil && isServerErr(c) {
			seg.Fault = true
		}
		//seg.Unlock()
		seg.Close(nil)
	}
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

func clientIP(r *http.Request) (string, bool) {
	forwardedFor := r.Header.Get("X-Forwarded-For")
	if forwardedFor != "" {
		return strings.TrimSpace(strings.Split(forwardedFor, ",")[0]), true
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr, false
	}
	return ip, false
}

//凡是调用了该函数，再去调用其他函数时，传递的都是sub ctx
func BeginSeg(ctxp *context.Context) guard.RecoverFunc {
	if ctxp == nil { //如果传入nil，则返回默认的保护函数
		return guard.DefaultRecover
	}
	var seg *xray.Segment
	if xray.GetSegment(*ctxp) == nil {
		*ctxp, seg = xray.BeginSegment(*ctxp, caller.Caller(2))
	} else {
		*ctxp, seg = xray.BeginSubsegment(*ctxp, caller.Caller(2))
	}
	return CloseSeg(seg)
}

//目前panic和*errp!=nil顶多发生一个
var CloseSeg = func(seg *xray.Segment) guard.RecoverFunc {
	return func(errp *error) {
		var err error
		if r := recover(); r != nil { //NOTE: 即使panic也要close
			err = fmt.Errorf("panic: %+v", r)
			//err = code.ServerErrPainc.WithErrorf("panic: %+v", r) //recover赋到*errp上，不再抛出panic
			if errp != nil {
				*errp = err
			}
		} else {
			if errp != nil {
				err = *errp
			}
		}
		seg.Close(err) //如果panic，这里可以记录到
		//FuncCounter.WithLabelValues(seg.Name).Inc()
		//FuncLatency.WithLabelValues(seg.Name).Observe((seg.EndTime - seg.StartTime) * 1000)
	}
}
