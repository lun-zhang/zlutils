package log

import (
	"github.com/sirupsen/logrus"
	"testing"
	"zlutils/caller"
)

func TestLog(t *testing.T) {
	caller.Init("zlutils")
	Init(Config{
		Level: 3,
	})
	logrus.Debugf("d")
	logrus.Infof("i")
	logrus.Error("e")
	logrus.Warnf("w")
}
