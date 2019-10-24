package xray

import (
	"bytes"
	"context"
	"github.com/aws/aws-xray-sdk-go/header"
	"github.com/aws/aws-xray-sdk-go/strategy/sampling"
	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net"
	"net/http"
	"strconv"
	"strings"
	"unicode"
	"zlutils/caller"
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
//NOTE: 如果ctxp==nil,则panic
func DoBeforeCtx(ctxp *context.Context) (args []interface{}) {
	var seg *xray.Segment
	source := NameReplace(caller.Caller(3))
	if xray.GetSegment(*ctxp) == nil {
		*ctxp, seg = xray.BeginSegment(*ctxp, source)
	} else {
		*ctxp, seg = xray.BeginSubsegment(*ctxp, source)
	}
	return []interface{}{seg}
}

//NOTE: 配套使用就不会有问题，否则panic
func DoAfter(err error, args ...interface{}) {
	seg := args[0].(*xray.Segment)
	seg.Close(err)
}

var nameValidSymbolMap = map[rune]struct{}{
	'_':  {},
	'.':  {},
	':':  {},
	'/':  {},
	'%':  {},
	'&':  {},
	'#':  {},
	'=':  {},
	'+':  {},
	'\\': {},
	'-':  {},
	'@':  {},
}

func nameRuneIsValid(r rune) bool {
	if unicode.IsDigit(r) {
		return true
	}
	if unicode.IsLetter(r) {
		return true
	}
	if unicode.IsSpace(r) {
		return true
	}
	if _, ok := nameValidSymbolMap[r]; ok {
		return true
	}
	return false
}

//替换成合法字符
// TODO: 如果用户觉得非法字符都替换成'-'不满意，可以自行修改
var NameReplace = func(name string) string {
	rs := make([]rune, len(name))
	for i, r := range name {
		if nameRuneIsValid(r) {
			rs[i] = r
		} else {
			rs[i] = '-'
		}
	}
	return string(rs)
}

/*跟踪ctxhttp.Do(ctx, xray.Client(client), request)发现发出请求时设置Header里的TraceId取自于seg.DownstreamHeader()：
/data/apps/go/pkg/mod/github.com/aws/aws-xray-sdk-go@v1.0.0-rc.5.0.20180720202646-037b81b2bf76/xray/segment_model.go 134行
*/
func GetTraceId(ctx context.Context) (traceId string) {
	if ctx == nil {
		return
	}
	seg := xray.GetSegment(ctx)
	if seg == nil {
		return
	}
	traceId = seg.TraceID
	if traceId != "" {
		return
	}
	parent := seg.ParentSegment
	if parent == nil {
		return
	}
	return parent.TraceID
}
