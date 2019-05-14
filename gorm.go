package zlutils

import (
	"context"
	"database/sql/driver"
	"fmt"
	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"reflect"
	"regexp"
	"time"
	"unicode"
)

var (
	sqlRegexp                = regexp.MustCompile(`\?`)
	numericPlaceHolderRegexp = regexp.MustCompile(`\$\d+`)
)

func isPrintable(s string) bool {
	for _, r := range s {
		if !unicode.IsPrint(r) {
			return false
		}
	}
	return true
}

type DB gorm.DB

func (db *DB) DB() *gorm.DB {
	return (*gorm.DB)(db)
}
func (db *DB) Begin() *DB {
	return (*DB)(db.DB().Begin())
}

//NOTE 只能用于初始化，失败则fatal
func InitDB(my MysqlConfig) *DB {
	var err error
	db, err := gorm.Open("mysql", my.Url)
	if err != nil {
		logrus.WithError(err).Fatal("mysql connect fail")
	}
	db.DB().SetMaxOpenConns(my.MaxOpenConns)
	db.DB().SetMaxIdleConns(my.MaxIdleConns)
	logrus.Info("mysql connect ok")
	return (*DB)(db)
}

type MysqlConfig struct {
	Url          string `json:"url"`
	MaxOpenConns int    `json:"max_open_conns"`
	MaxIdleConns int    `json:"max_idle_conns"`
}

func (db *DB) BeginSubsegment(ctx context.Context) (clone *gorm.DB) {
	source := GetSource(2)
	_, seg := xray.BeginSubsegment(ctx, "mysql-"+source) //begin segment
	seg.Namespace = "remote"

	clone = db.DB().Model(db.DB().Value) //NOTE: 必须返回一个clone后的，否则两个db使用相同的logger打印错误
	clone.LogMode(true).
		SetLogger(&dbLogger{
			ctx:    ctx,
			seg:    seg,
			source: source,
		})
	return clone
}

type dbLogger struct { //NOTE: private只能被BeginSubsegment用
	ctx    context.Context
	seg    *xray.Segment
	source string
}

//NOTE: 打印到logrus、trace、xray
func (l dbLogger) Print(values ...interface{}) {
	if len(values) <= 1 {
		return
	}

	entry := logrus.WithFields(logrus.Fields{
		"stack":  nil,
		"source": l.source,
	})

	if level := values[0]; level == "sql" { //NOTE: error时会打两条日志，先进入下面的else再进入这里的if
		// duration
		duration := values[2].(time.Duration)
		// sql
		sql := getSQL(values[3].(string), values[4])
		l.seg.GetSQL().SanitizedQuery = sql
		l.seg.Close(nil) //FIXME: 是否会遇到无法关闭或重复关闭的情况

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
func getSQL(base string, vars interface{}) (sql string) {
	var formattedValues []string
	for _, value := range vars.([]interface{}) {
		indirectValue := reflect.Indirect(reflect.ValueOf(value))
		if indirectValue.IsValid() {
			value = indirectValue.Interface()
			if t, ok := value.(time.Time); ok {
				formattedValues = append(formattedValues, fmt.Sprintf("'%v'", t.Format("2006-01-02 15:04:05")))
			} else if b, ok := value.([]byte); ok {
				if str := string(b); isPrintable(str) {
					formattedValues = append(formattedValues, fmt.Sprintf("'%v'", str))
				} else {
					formattedValues = append(formattedValues, "'<binary>'")
				}
			} else if r, ok := value.(driver.Valuer); ok {
				if value, err := r.Value(); err == nil && value != nil {
					formattedValues = append(formattedValues, fmt.Sprintf("'%v'", value))
				} else {
					formattedValues = append(formattedValues, "NULL")
				}
			} else {
				formattedValues = append(formattedValues, fmt.Sprintf("'%v'", value))
			}
		} else {
			formattedValues = append(formattedValues, "NULL")
		}
	}

	// differentiate between $n placeholders or else treat like ?
	if numericPlaceHolderRegexp.MatchString(base) {
		sql = base
		for index, value := range formattedValues {
			placeholder := fmt.Sprintf(`\$%d([^\d]|$)`, index+1)
			sql = regexp.MustCompile(placeholder).ReplaceAllString(sql, value+"$1")
		}
	} else {
		formattedValuesLength := len(formattedValues)
		for index, value := range sqlRegexp.Split(base, -1) {
			sql += value
			if index < formattedValuesLength {
				sql += formattedValues[index]
			}
		}
	}
	return
}

//NOTE: 提交或回滚，如果失败则写到*errp中
func (db *DB) CloseTransaction(ctx context.Context, errp *error) {
	defer BeginSubsegment(&ctx)()

	if *errp != nil {
		if err := db.DB().Rollback().Error; err != nil {
			logrus.WithFields(logrus.Fields{
				"error":          (*errp).Error(),
				"rollback_error": err.Error(),
			}).Error("rollback fail")
			*errp = err
		}
	} else {
		if err := db.DB().Commit().Error; err != nil {
			logrus.WithField("commit_error", err.Error()).Error("commit fail")
			*errp = err
		}
	}
}

func GetMethod(sql string) (method string) {
	if _, err := fmt.Sscanf(sql, "%s", &method); err != nil {
		return unknown
	}
	return
}
