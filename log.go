package zlutils

import (
	"fmt"
	"github.com/lestrrat/go-file-rotatelogs"
	"github.com/rifflock/lfshook"
	"github.com/sirupsen/logrus"
	"os"
	"time"
)

type MyFormatter struct {
	logrus.Formatter
}

func (f MyFormatter) Format(e *logrus.Entry) ([]byte, error) {
	if e.Level != logrus.InfoLevel {
		//获取位置影响性能，所以忽略info，如果设置打印级别高于debug，则Debug()不会调用这里
		//使用stack而不是source，因为skip会发生变化
		//stack的skip=3是为了忽略这几个函数Format GetStack GetSource
		if _, ok := e.Data["stack"]; !ok {
			//允许外部记录stack，而不覆盖
			e.Data["stack"] = GetStack(3)
		}
	}
	e.Time = e.Time.UTC() //改成UTC时间
	return f.Formatter.Format(e)
}

func getLogWriter(level string) *rotatelogs.RotateLogs {
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
		//最多7个文件，配合每天分割文件，则是每7天删除旧日志
		rotatelogs.WithMaxAge(-1),       //默认7*24h，配合次数时需显式设为-1关闭
		rotatelogs.WithRotationCount(7), //默认-1
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

	formatter := MyFormatter{&logrus.JSONFormatter{}}
	logrus.SetFormatter(formatter) //用于屏幕输出格式化

	debugWriter := getLogWriter("debug")
	infoWriter := getLogWriter("info")
	warnWriter := getLogWriter("warn")
	errorWriter := getLogWriter("error")
	//高级别的日志同时也写到低级别的文件中
	logrus.AddHook(lfshook.NewHook(
		lfshook.WriterMap{
			logrus.FatalLevel: errorWriter,
			logrus.PanicLevel: errorWriter,
			logrus.ErrorLevel: errorWriter,
		},
		formatter,
	))
	logrus.AddHook(lfshook.NewHook(
		lfshook.WriterMap{
			logrus.FatalLevel: warnWriter,
			logrus.PanicLevel: warnWriter,
			logrus.ErrorLevel: warnWriter,
			logrus.WarnLevel:  warnWriter,
		},
		formatter,
	))
	logrus.AddHook(lfshook.NewHook(
		lfshook.WriterMap{
			logrus.FatalLevel: infoWriter,
			logrus.PanicLevel: infoWriter,
			logrus.ErrorLevel: infoWriter,
			logrus.WarnLevel:  infoWriter,
			logrus.InfoLevel:  infoWriter,
		},
		formatter,
	))
	logrus.AddHook(lfshook.NewHook(
		lfshook.WriterMap{
			logrus.FatalLevel: debugWriter,
			logrus.PanicLevel: debugWriter,
			logrus.ErrorLevel: debugWriter,
			logrus.WarnLevel:  debugWriter,
			logrus.InfoLevel:  debugWriter,
			logrus.DebugLevel: debugWriter,
		},
		formatter,
	))
}
