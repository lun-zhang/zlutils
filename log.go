package zlutils

import (
	"github.com/sirupsen/logrus"
	"os"
)

type MyFormatter struct {
	logrus.JSONFormatter
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
	e.Time = e.Time.UTC()               //改成UTC时间
	e.Data["time_unix"] = e.Time.Unix() //方便grep查询范围
	serialized, err = f.JSONFormatter.Format(e)
	return
}

func InitLog(release bool) {
	if !release {
		logrus.SetLevel(logrus.DebugLevel) //默认info
	}

	logrus.SetFormatter(MyFormatter{})
	logrus.SetOutput(os.Stdout)
}
