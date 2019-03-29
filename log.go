package zlutils

import (
	"fmt"
	"github.com/lestrrat/go-file-rotatelogs"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"os"
	"time"
)

type MyFormatter struct {
	logrus.JSONFormatter
	WriterMap map[logrus.Level]io.Writer
}

func (f MyFormatter) Format(e *logrus.Entry) (serialized []byte, err error) {
	LogCounter.WithLabelValues(e.Level.String()).Inc()
	if e.Level != logrus.InfoLevel {
		if stack, ok := e.Data["stack"]; !ok {
			e.Data["stack"] = GetStack(3) //允许外部记录stack，而不覆盖
		} else if stack == nil {
			delete(e.Data, "stack") //如果被置为nil则不输出
		}
	}
	e.Time = e.Time.UTC() //改成UTC时间
	serialized, err = f.JSONFormatter.Format(e)
	if err != nil {
		return
	}
	err = f.write(e.Level, serialized)
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
	dir := fmt.Sprintf("/data/logs/%s", ProjectName)
	path := fmt.Sprintf("%s/%s.log", dir, level)
	if _, err := os.Stat(dir); err != nil || os.IsNotExist(err) {
		panic(fmt.Errorf("not exist dir %s", dir))
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
		rotatelogs.WithMaxAge(-1),       //默认7*24h，配合次数时需显式设为-1关闭
		rotatelogs.WithRotationCount(3), //默认-1
	)
	if err != nil {
		panic(err)
	}
	return writer
}

func InitLog(release bool) {
	if !release {
		logrus.SetLevel(logrus.DebugLevel) //默认info
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
}
