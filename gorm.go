package zlutils

import (
	"context"
	"database/sql/driver"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"reflect"
	"regexp"
	"strconv"
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

func InitGormLogFormatter() {
	gorm.LogFormatter = func(values ...interface{}) (messages []interface{}) {
		if len(values) > 1 {
			var (
				sql             string
				formattedValues []string
				level           = values[0]
				currentTime     = "\n\033[33m[" + gorm.NowFunc().Format("2006-01-02 15:04:05") + "]\033[0m"
				source          = fmt.Sprintf("\033[35m(%v)\033[0m", values[1])
			)

			messages = []interface{}{source, currentTime}
			entry := logrus.WithFields(logrus.Fields{
				"stack":  nil,
				"source": values[1],
			})
			//error时会打两条日志，先进入下面的else再进入if
			if level == "sql" {
				// duration
				duration := values[2].(time.Duration)
				messages = append(messages, fmt.Sprintf(" \033[36;1m[%.2fms]\033[0m ", float64(duration.Nanoseconds()/1e4)/100.0))
				entry = entry.WithField("duration", fmt.Sprintf("%.2fms", float64(duration.Nanoseconds()/1e4)/100.0))
				// sql

				for _, value := range values[4].([]interface{}) {
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
				if numericPlaceHolderRegexp.MatchString(values[3].(string)) {
					sql = values[3].(string)
					for index, value := range formattedValues {
						placeholder := fmt.Sprintf(`\$%d([^\d]|$)`, index+1)
						sql = regexp.MustCompile(placeholder).ReplaceAllString(sql, value+"$1")
					}
				} else {
					formattedValuesLength := len(formattedValues)
					for index, value := range sqlRegexp.Split(values[3].(string), -1) {
						sql += value
						if index < formattedValuesLength {
							sql += formattedValues[index]
						}
					}
				}

				messages = append(messages, sql)
				messages = append(messages, fmt.Sprintf(" \n\033[36;31m[%v]\033[0m ", strconv.FormatInt(values[5].(int64), 10)+" rows affected or returned "))
				entry = entry.WithFields(logrus.Fields{
					"sql": sql,
					"rows affected or returned": values[5],
				})
				if duration >= 200*time.Millisecond {
					//慢查询警告
					entry.Warn("duration bigger than 200ms")
				} else {
					entry.Debug()
				}
				method := GetMethod(sql)
				MysqlCounter.WithLabelValues(method).Inc()
				MysqlLatency.WithLabelValues(method).Observe(duration.Seconds() * 1000)
			} else {
				messages = append(messages, "\033[31;1m")
				messages = append(messages, values[2:]...)
				messages = append(messages, "\033[0m")
				entry.Debug(values[2:]) //error时候会到这里，不打error日志，让外面去打，因为外面还有其他信息
			}
		}
		return
	}
}

func CloseTransaction(ctx context.Context, tx *gorm.DB, errp *error) {
	defer BeginSubsegment(&ctx)()

	if *errp != nil {
		if err := tx.Rollback().Error; err != nil {
			logrus.WithFields(logrus.Fields{
				"error":          (*errp).Error(),
				"rollback_error": err.Error(),
			}).Error("rollback fail")
			*errp = err
		}
	} else {
		if err := tx.Commit().Error; err != nil {
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
