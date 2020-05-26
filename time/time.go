package time

import (
	"database/sql/driver"
	"fmt"
	"strconv"
	"time"
)

//NOTE: 让数据库能使用该类型，但是存数据库时候不会存成json
type Time struct {
	time.Time
}

func Now() Time {
	return Time{Time: time.Now()}
}

func (t *Time) UnmarshalJSON(b []byte) error {
	s := string(b)
	if s == "null" {
		//NOTE: 依照官方做法no-op
		return nil
	}

	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}

	t.Time = time.Unix(i, 0)
	return nil
}

func (t Time) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatInt(t.Unix(), 10)), nil
}

func (t Time) Value() (driver.Value, error) {
	return t.Time, nil
}

func (t *Time) Scan(src interface{}) error {
	tmp, ok := src.(time.Time)
	if !ok {
		return fmt.Errorf("src must be time.Time")
	}
	t.Time = tmp
	return nil
}

// Deprecated: 该函数将要删掉，请换用 GetZoneZeroUTC
func GetIndianZeroUTC(t time.Time) time.Time {
	t = t.UTC()
	t1830 := time.Date(t.Year(), t.Month(), t.Day(), 18, 30, 0, 0, time.UTC)
	if t.Sub(t1830) < 0 {
		t1830 = t1830.Add(-24 * time.Hour)
	}
	return t1830
}

//相对UTC的时区偏移量，例如
// 中国是"8h"
// 印度是"5h30m"
// 印尼是"7h"
// 华盛顿是"-4h"
// 默认是印度
//推荐用consul初始化
var ZoneOffset = Duration{Duration: 5*time.Hour + 30*time.Minute}

//获得ZoneOffset代表的时区对应的零点，转成UTC
func GetZoneZeroUTC(t time.Time) time.Time {
	t = t.UTC()
	t = t.Add(ZoneOffset.Duration) //先UTC加上时区，当做带时区的时间，这样去计算年月日是正确的
	t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	return t.Add(-ZoneOffset.Duration) //最后减去时区，就是UTC时间
}

type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalText(text []byte) (err error) {
	d.Duration, err = time.ParseDuration(string(text))
	return
}
