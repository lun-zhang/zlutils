package zlutils

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"time"

	"github.com/lun-zhang/gorm"
)

type DBLogger struct{}

//NOTE: 打印到logrus、trace、xray
func (l DBLogger) Print(values ...interface{}) {
	if len(values) <= 1 {
		return
	}

	entry := logrus.WithFields(logrus.Fields{
		"stack":  nil,
		"source": values[1],
	})

	if level := values[0]; level == "sql" { //NOTE: error时会打两条日志，先进入下面的else再进入这里的if
		// duration
		duration := values[2].(time.Duration)
		// sql
		sql := gorm.PrintSQL(values[3].(string), values[4].([]interface{})...)

		entry = entry.WithFields(logrus.Fields{
			"sql": sql,
			"rows affected or returned": values[5],
			"duration":                  duration.String(),
		})
		if duration >= 200*time.Millisecond {
			entry.Warn("slow sql") //慢查询警告
		} else {
			entry.Debug()
		}
		method := GetMethod(sql)
		MysqlCounter.WithLabelValues(method).Inc()
		MysqlLatency.WithLabelValues(method).Observe(duration.Seconds() * 1000)
	} else {
		entry.Debug(values[2:]) //NOTE: error时候会先到这里，不打error日志，让外面去打，因为外面还有其他信息
	}
}

func GetMethod(sql string) (method string) {
	if _, err := fmt.Sscanf(sql, "%s", &method); err != nil {
		return unknown
	}
	return
}

//NOTE 只能用于初始化，失败则fatal
func GetDB(my MysqlConfig) *gorm.DB {
	entry := logrus.WithField("my", my)
	var err error
	db, err := gorm.Open("mysql", my.Url)
	if err != nil {
		entry.WithError(err).Fatal("mysql connect fail")
	}
	db.DB().SetMaxOpenConns(my.MaxOpenConns)
	db.DB().SetMaxIdleConns(my.MaxIdleConns)
	db.LogMode(true).SetLogger(&DBLogger{})
	entry.Info("mysql connect ok")
	return db
}

type MysqlConfig struct {
	Url          string `json:"url"`
	MaxOpenConns int    `json:"max_open_conns"`
	MaxIdleConns int    `json:"max_idle_conns"`
}