package zlutils

import (
	"fmt"
	"strconv"
	"time"
)

type JsonTime struct {
	time.Time
}

func (t *JsonTime) UnmarshalJSON(b []byte) error {
	s := string(b)
	if s == "null" {
		t.Time = time.Time{}
		return nil
	}

	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}

	t.Time = time.Unix(i, 0)
	return nil
}

func (t JsonTime) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%s", strconv.FormatInt(t.Time.Unix(), 10))), nil
}
