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

func GetIndianZeroUTC(t time.Time) time.Time {
	t = t.UTC()
	t1830 := time.Date(t.Year(), t.Month(), t.Day(), 18, 30, 0, 0, time.UTC)
	if t.Sub(t1830) < 0 {
		t1830 = t1830.Add(-24 * time.Hour)
	}
	return t1830
}

type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalText(text []byte) (err error) {
	d.Duration, err = time.ParseDuration(string(text))
	return
}
