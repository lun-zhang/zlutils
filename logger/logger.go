package logger

import (
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/lestrrat/go-file-rotatelogs"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"os"
	"time"
	"zlutils/caller"
)

type MyFormatter struct {
	logrus.JSONFormatter
	WriterMap map[logrus.Level]io.Writer
}

func (f MyFormatter) Format(e *logrus.Entry) (serialized []byte, err error) {
	//LogCounter.WithLabelValues(e.Level.String()).Inc()
	if e.Level != logrus.InfoLevel {
		if stack, ok := e.Data["stack"]; !ok {
			e.Data["stack"] = caller.Stack(3) //允许外部记录stack，而不覆盖
		} else if stack == nil {
			delete(e.Data, "stack") //如果被置为nil则不输出
		}
	}
	e.Time = e.Time.UTC()               //改成UTC时间
	e.Data["time_unix"] = e.Time.Unix() //方便grep查询范围
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
		rotatelogs.WithMaxAge(-1),                          //默认7*24h，配合次数时需显式设为-1关闭
		rotatelogs.WithRotationCount(output.RotationCount), //默认-1
	)
	if err != nil {
		panic(err)
	}
	return writer
}

type Config struct {
	Level  logrus.Level `json:"level"`
	Output *Output      `json:"output"`
}
type Output struct { //如果nil则输出到屏幕
	Dir           string `json:"dir"`
	RotationCount int    `json:"rotation_count"`
}

var output Output

func Init(config Config) {
	logrus.SetLevel(config.Level)
	if config.Output != nil {
		output = *config.Output
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
	logId int64
}

//NOTE: 请求和响应会打两条日志，响应的时候会把请求放在一起再打印一遍，可能会觉得冗余，但好处是完整
func (w debugWriter) Write(b []byte) (n int, err error) {
	logrus.WithFields(logrus.Fields{
		"log_id":        w.logId,
		"stack":         nil,
		"response_body": string(b),
	}).Debug()
	return w.ResponseWriter.Write(b)
}

//NOTE: 上线后的接口不该使用这个中间件，对性能有影响
func MidDebug() gin.HandlerFunc {
	return func(c *gin.Context) {
		if logrus.IsLevelEnabled(logrus.DebugLevel) { //这样方便watch level
			buf := new(bytes.Buffer)
			buf.ReadFrom(c.Request.Body)
			reqBody := buf.Bytes()
			logId := time.Now().UnixNano()
			logrus.WithFields(logrus.Fields{
				//TODO: 完善字段
				"log_id":       logId,
				"path":         c.Request.URL.Path,
				"method":       c.Request.Method,
				"header":       c.Request.Header,
				"request_body": string(reqBody),
				"stack":        nil,
			}).Debug()
			c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(reqBody)) //拿出来再放回去
			c.Writer = debugWriter{
				ResponseWriter: c.Writer,
				logId:          logId,
			}
		}
	}
}
