package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/lestrrat/go-file-rotatelogs"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
	"zlutils/caller"
	"zlutils/consul"
	"zlutils/xray"
)

type MyFormatter struct {
	logrus.JSONFormatter
	WriterMap map[logrus.Level]io.Writer
}

const (
	FieldStack    = "stack"
	FieldTraceId  = "trace_id"
	FieldTimeUnix = "time_unix"
)

//公共域，优先级低于常规打日志时候的域
//TODO: 像是level这样不在Data里的后续再看
var comFields = logrus.Fields{}

func SetComField(key string, value interface{}) {
	comFields[key] = value
}

func SetComFields(fields logrus.Fields) {
	for k, v := range fields {
		SetComField(k, v)
	}
}

func (f MyFormatter) Format(e *logrus.Entry) (serialized []byte, err error) {
	if MetricCounter != nil {
		MetricCounter(e).Inc()
	}
	if e.Level != logrus.InfoLevel {
		if stack, ok := e.Data[FieldStack]; !ok {
			e.Data[FieldStack] = caller.Stack(3) //允许外部记录stack，而不覆盖
		} else if stack == nil {
			delete(e.Data, FieldStack) //如果被置为nil则不输出
		}
	}
	e.Time = e.Time.UTC()                 //改成UTC时间
	e.Data[FieldTimeUnix] = e.Time.Unix() //方便grep查询范围

	traceId := xray.GetTraceId(e.Context)
	if traceId != "" {
		e.Data[FieldTraceId] = traceId
	}

	//填充公共域
	for k, v := range comFields {
		if _, ok := e.Data[k]; !ok {
			e.Data[k] = v
		}
	}

	serialized, err = f.JSONFormatter.Format(e)
	if err != nil {
		return
	}
	if f.WriterMap != nil {
		err = f.write(e.Level, serialized)
	} //else输出到屏幕
	return
}

func (f MyFormatter) write(level logrus.Level, serialized []byte) (err error) {
	if writer := f.WriterMap[level]; writer != nil {
		if _, err = writer.Write(serialized); err != nil {
			return
		}
	}
	//debug模式
	if logrus.IsLevelEnabled(logrus.DebugLevel) {
		//输出到屏幕
		os.Stderr.Write(serialized)
		if level != logrus.DebugLevel {
			//其他日志同时输出到debug.log
			if writer := f.WriterMap[logrus.DebugLevel]; writer != nil {
				if _, err = writer.Write(serialized); err != nil {
					return
				}
			}
		}
	}
	return
}

func getLogWriter(level logrus.Level) *rotatelogs.RotateLogs {
	path := fmt.Sprintf("%s/%s.log", output.Dir, level)
	if _, err := os.Stat(output.Dir); err != nil || os.IsNotExist(err) {
		panic(fmt.Errorf("not exist dir %s", output.Dir))
	}
	writer, err := rotatelogs.New(
		path+".%Y%m%d",
		rotatelogs.WithLinkName(path),
		//修改为地区和时钟为UTC，用于日志回收、命名等操作中用到时间的地方
		rotatelogs.WithLocation(time.UTC),
		rotatelogs.WithClock(rotatelogs.UTC), //默认loc，改成UTC
		//每天分割
		rotatelogs.WithRotationTime(time.Hour*24), //默认24h
		//最多3个文件，配合每天分割文件，则是每3天删除旧日志
		rotatelogs.WithMaxAge(-1),                //默认7*24h，配合次数时需显式设为-1关闭
		rotatelogs.WithRotationCount(output.Rot), //默认-1
	)
	if err != nil {
		panic(err)
	}
	return writer
}

type Config struct {
	Level  logrus.Level `json:"level"`  //Level自带类型检查
	Output *Output      `json:"output"` //如果nil则输出到屏幕
}
type Output struct {
	Dir string `json:"dir"` //目录
	Rot int    `json:"rot"` //默认为3，表示日志最多保留rot天
}

var defaultConfig = Config{Level: logrus.InfoLevel}

var output Output

func InitByConsul(key string) {
	var config Config
	consul.GetJson(key, &config)
	Init(config)
}
func WatchByConsul(key string) {
	consul.WatchJsonVarious(key, func(config Config) {
		Init(config)
	})
}

func init() {
	Init(defaultConfig)
}

func Init(config Config) {
	logrus.SetLevel(config.Level)
	if config.Output != nil {
		output = *config.Output
		if output.Rot == 0 {
			output.Rot = 3
		}
		errorWriter := getLogWriter(logrus.ErrorLevel)
		logrus.SetFormatter(MyFormatter{
			WriterMap: map[logrus.Level]io.Writer{
				logrus.FatalLevel: errorWriter,
				logrus.PanicLevel: errorWriter,
				logrus.ErrorLevel: errorWriter,
				logrus.WarnLevel:  getLogWriter(logrus.WarnLevel),
				logrus.InfoLevel:  getLogWriter(logrus.InfoLevel),
				logrus.DebugLevel: getLogWriter(logrus.DebugLevel),
			},
		})
		logrus.SetOutput(ioutil.Discard)
	} else {
		logrus.SetFormatter(MyFormatter{})
		logrus.SetOutput(os.Stdout)
	}
}

type debugWriter struct {
	gin.ResponseWriter
	traceId string
}

func tryGetJson(header http.Header, b []byte) (resp interface{}) {
	if strings.Contains(header.Get("Content-Type"), "application/json") {
		if er := json.Unmarshal(b, &resp); er == nil {
			return
		}
	}
	return string(b) //FIXME: 不会用非打印字符吧
}

//NOTE: 请求和响应会打两条日志，响应的时候会把请求放在一起再打印一遍，可能会觉得冗余，但好处是完整
func (w debugWriter) Write(b []byte) (n int, err error) {
	logrus.WithFields(logrus.Fields{
		FieldTraceId:    w.traceId,
		FieldStack:      nil,
		"response_body": tryGetJson(w.Header(), b),
	}).Debug()
	return w.ResponseWriter.Write(b)
}

//NOTE: 上线后日志级别当高于debug，对性能有影响
func MidDebug() gin.HandlerFunc {
	return func(c *gin.Context) {
		if logrus.IsLevelEnabled(logrus.DebugLevel) { //这样方便watch level
			buf := new(bytes.Buffer)
			buf.ReadFrom(c.Request.Body)
			reqBody := buf.Bytes()
			traceId := xray.GetTraceId(c.Request.Context())
			logrus.WithFields(logrus.Fields{
				//TODO: 完善字段
				FieldTraceId:   traceId,
				FieldStack:     nil,
				"host":         c.Request.Host,
				"path":         c.Request.URL.Path,
				"query":        c.Request.URL.Query(),
				"method":       c.Request.Method,
				"header":       c.Request.Header,
				"request_body": tryGetJson(c.Request.Header, reqBody),
				"remote_addr":  c.Request.RemoteAddr,
				"client_ip":    c.ClientIP(),
			}).Debug()
			c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(reqBody)) //拿出来再放回去
			c.Writer = debugWriter{
				ResponseWriter: c.Writer,
				traceId:        traceId,
			}
		}
	}
}

func MidInfo() gin.HandlerFunc {
	return func(c *gin.Context) {
		if logrus.IsLevelEnabled(logrus.InfoLevel) {
			start := time.Now()
			c.Next()
			logrus.WithContext(c.Request.Context()).WithFields(logrus.Fields{
				"statusCode": c.Writer.Status(),
				"latency":    time.Now().Sub(start).String(),
				"clientIP":   c.ClientIP(),
				"endpoint":   fmt.Sprintf("%s-%s", c.Request.URL.Path, c.Request.Method),
			}).Info()
		}
	}
}
