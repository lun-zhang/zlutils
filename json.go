package zlutils

import (
	"strconv"
	"time"
)

type JsonTime time.Time //NOTE: 不是与数据库的转换，是与客户端的转换

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
