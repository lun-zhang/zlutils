package mysql

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/lun-zhang/gorm"
	"github.com/sirupsen/logrus"
	"reflect"
	"strings"
	"time"
)

//虽然这个Logger也可以打印github.com/jinzhu/gorm的日志，但是没有ctx所以无法加入trace_id，
// 因此github.com/lun-zhang/gorm v1.13.1直接打印了日志，有tract_id
type Logger struct{}

//NOTE: 打印到logrus、trace、xray
func (Logger) Print(values ...interface{}) {
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
		args := values[4].([]interface{})
		sql := gorm.PrintSQL(query, args...)

		entry = entry.WithFields(logrus.Fields{
			"sql":      sql,
			"rows":     values[5], //rows affected or returned
			"duration": duration.String(),
		})
		if duration >= 200*time.Millisecond {
			entry.Warn("slow sql") //慢查询警告
		} else {
			entry.Debug()
		}
		if MetricCounter != nil {
			MetricCounter(query, args...).Inc()
		}
		if MetricLatency != nil {
			MetricLatency(query, args...).Observe(duration.Seconds() * 1000)
		}
	} else {
		entry.Debug(values[2:]) //NOTE: error时候会先到这里，不打error日志，让外面去打，因为外面还有其他信息
	}
}

//只监控不日志，日志已放到github.com/lun-zhang/gorm中
type metricLogger struct{}

func (metricLogger) Print(values ...interface{}) {
	if len(values) <= 1 {
		return
	}

	if level := values[0]; level == "sql" {
		// duration
		duration := values[2].(time.Duration)
		// sql
		query := values[3].(string)
		args := values[4].([]interface{})
		if MetricCounter != nil {
			MetricCounter(query, args...).Inc()
		}
		if MetricLatency != nil {
			MetricLatency(query, args...).Observe(duration.Seconds() * 1000)
		}
	}
}

//NOTE 只能用于初始化，失败则fatal
func New(config Config) *gorm.DB {
	entry := logrus.WithField("config", config)
	db, err := gorm.Open("mysql", config.Url)
	if err != nil {
		entry.WithError(err).Fatal("mysql connect fail")
	}
	db.DB().SetMaxOpenConns(config.MaxOpenConns)
	db.DB().SetMaxIdleConns(config.MaxIdleConns)
	db.LogMode(true).SetLogger(&metricLogger{})
	entry.Info("mysql connect ok")
	return db
}

func NewMasterAndSlave(config ConfigMasterAndSlave) *gorm.DB {
	entry := logrus.WithField("config", config)
	db, err := gorm.OpenMasterAndSlave("mysql", config.Master.Url, config.Slave.Url)
	if err != nil {
		entry.WithError(err).Fatal("mysql connect fail")
	}
	config.Master.setConns(db.DB())
	config.Slave.setConns(db.DBSlave())
	db.LogMode(true).SetLogger(&metricLogger{})
	entry.Info("mysql connect ok")
	return db
}

type Config struct {
	Url          string `json:"url"`
	MaxOpenConns int    `json:"max_open_conns"`
	MaxIdleConns int    `json:"max_idle_conns"`
}

func (my Config) setConns(db *sql.DB) {
	db.SetMaxOpenConns(my.MaxOpenConns)
	db.SetMaxIdleConns(my.MaxIdleConns)
}

type ConfigMasterAndSlave struct {
	Master Config `json:"master"`
	Slave  Config `json:"slave"`
}

//获取gorm列，忽略omits指定的列，主要用于解决Save不存在会插入、Updates不会更新更新零值的问题
func OmitColumns(value interface{}, omits ...string) (m map[string]interface{}) {
	ignore := map[string]struct{}{}
	for _, o := range omits {
		ignore[o] = struct{}{}
	}

	m = map[string]interface{}{}
	omitColumns(reflect.ValueOf(value), ignore, m)
	return
}

func omitColumns(v reflect.Value, ignore map[string]struct{}, m map[string]interface{}) {
	if v.Kind() != reflect.Struct {
		return
	}
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		vi := v.Field(i)
		ti := t.Field(i)
		if ti.Anonymous {
			omitColumns(vi, ignore, m)
		} else {
			if name, ok := getGormColumnName(ti); ok {
				if _, ok := ignore[name]; !ok {
					m[name] = vi.Interface()
				}
			}
		}
	}
}

//简单处理：定义了column了才是有效的
func getGormColumnName(ti reflect.StructField) (string, bool) {
	if n := ti.Name[0]; !(n >= 'A' && n <= 'Z') { //非导出的却有gorm标签的
		return "", false
	}
	tag := ti.Tag.Get("gorm")
	for _, p := range strings.Split(tag, ";") {
		if strings.HasPrefix(p, "column:") {
			name := p[len("column:"):]
			if name == "" {
				return "", false
			}
			return name, true
		}
	}
	return "", false
}
