package db

import (
	"github.com/sirupsen/logrus"
	"time"

	"github.com/lun-zhang/gorm"
)

type Logger struct{}

//NOTE: 打印到logrus、trace、xray
func (l Logger) Print(values ...interface{}) {
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
		query := values[3].(string)
		sql := gorm.PrintSQL(query, values[4].([]interface{})...)

		entry = entry.WithFields(logrus.Fields{
			"sql":                       sql,
			"rows affected or returned": values[5],
			"duration":                  duration.String(),
		})
		if duration >= 200*time.Millisecond {
			entry.Warn("slow sql") //慢查询警告
		} else {
			entry.Debug()
		}
		//query = strings.Replace(query, "?,", "", -1) //FIXME 改成更好的做法，把IN(?,...)替换成IN(...)
		//MysqlCounter.WithLabelValues(query).Inc()
		//MysqlLatency.WithLabelValues(query).Observe(duration.Seconds() * 1000)
	} else {
		entry.Debug(values[2:]) //NOTE: error时候会先到这里，不打error日志，让外面去打，因为外面还有其他信息
	}
}

//NOTE 只能用于初始化，失败则fatal
func NewDB(config Config) *gorm.DB {
	entry := logrus.WithField("config", config)
	db, err := gorm.Open("mysql", config.Url)
	if err != nil {
		entry.WithError(err).Fatal("mysql connect fail")
	}
	db.DB().SetMaxOpenConns(config.MaxOpenConns)
	db.DB().SetMaxIdleConns(config.MaxIdleConns)
	db.LogMode(true).SetLogger(&Logger{})
	entry.Info("mysql connect ok")
	return db
}

func NewDBMasterAndSlave(config ConfigMasterAndSlave) *gorm.DB {
	entry := logrus.WithField("config", config)
	db, err := gorm.OpenMasterAndSlave("mysql", config.Master.Url, config.Slave.Url)
	if err != nil {
		entry.WithError(err).Fatal("mysql connect fail")
	}
	config.Master.setConns(db)
	config.Slave.setConns(db)
	db.LogMode(true).SetLogger(&Logger{})
	entry.Info("mysql connect ok")
	return db
}

type Config struct {
	Url          string `json:"url"`
	MaxOpenConns int    `json:"max_open_conns"`
	MaxIdleConns int    `json:"max_idle_conns"`
}

func (my Config) setConns(db *gorm.DB) {
	db.DB().SetMaxOpenConns(my.MaxOpenConns)
	db.DB().SetMaxIdleConns(my.MaxIdleConns)
}

type ConfigMasterAndSlave struct {
	Master Config `json:"master"`
	Slave  Config `json:"slave"`
}
