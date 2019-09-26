package meta

import (
	"github.com/sirupsen/logrus"
	"xlbj-gitlab.xunlei.cn/oversea/zlutils/v7/caller"
)

type Meta map[string]interface{}

//似乎实现了MustGet Set的都可以当做meta
func (m Meta) MustGet(k string) interface{} {
	entry := logrus.WithFields(logrus.Fields{
		"caller": caller.Caller(2),
		"m":      m,
	})
	if m == nil {
		entry.Panic("m is nil")
	}
	v, ok := (m)[k]
	if !ok {
		entry.Panic("no value")
	}
	return v
}
func (m *Meta) Set(k string, v interface{}) {
	if *m == nil { //懒加载
		*m = Meta{}
	}
	(*m)[k] = v
}
