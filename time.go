package zlutils

import (
	"database/sql/driver"
	"fmt"
	"strconv"
	"time"
)

type JsonTime time.Time //NOTE: 让数据库能使用该类型，但是存数据库时候不会存成json

//方便打印
func (t JsonTime) String() string {
	return time.Time(t).String()
}

func (t *JsonTime) UnmarshalJSON(b []byte) error {
	s := string(b)
	if s == "null" {
		//NOTE: 依照官方做法no-op
		return nil
	}

	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}

	*t = JsonTime(time.Unix(i, 0))
	return nil
}

func (t JsonTime) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatInt(time.Time(t).Unix(), 10)), nil
}

func (t JsonTime) Value() (driver.Value, error) {
	return time.Time(t), nil
}

func (t *JsonTime) Scan(src interface{}) error {
	tmp, ok := src.(time.Time)
	if !ok {
		return fmt.Errorf("JsonTime src must be time.Time")
	}
	*t = JsonTime(tmp)
	return nil
}

func GetIndianZeroUTC(t time.Time) time.Time {
	t = t.UTC()
	t1830 := time.Date(t.Year(), t.Month(), t.Day(), 18, 30, 0, 0, time.UTC)
	if t.Sub(t1830) < 0 {
		t1830 = t1830.Add(-24 * time.Hour)
	}
	return t1830
}
