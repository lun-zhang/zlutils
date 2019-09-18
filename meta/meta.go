package meta

import (
	"github.com/sirupsen/logrus"
)

type Meta map[string]interface{}

//似乎实现了MustGet Set的都可以当做meta
func (m Meta) MustGet(k string) interface{} {
	if m == nil {
		logrus.Fatal("m is nil")
	}
	v, ok := (m)[k]
	if !ok {
		logrus.WithField("m", m).Fatalf("no key:%s", k)
	}
	return v
}
func (m *Meta) Set(k string, v interface{}) {
	if *m == nil { //懒加载
		*m = Meta{}
	}
	(*m)[k] = v
}
