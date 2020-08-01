package dtalk

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"testing"
	"time"
)

func TestAsyncAlertError(t *testing.T) {
	Init(map[logrus.Level]string{
		logrus.ErrorLevel: "5dbf8117b839d8335462cd2807898b7a5e148fe98d88815fdcc60c12704c2855",
		logrus.WarnLevel:  "5dbf8117b839d8335462cd2807898b7a5e148fe98d88815fdcc60c12704c2855",
	})

	type tmp struct {
		A int
	}
	//SetComHeadLines("head")
	SetComTailLines(fmt.Sprintf("pid:%d", os.Getpid()))

	AsyncAlert(logrus.ErrorLevel, "调试-错误", "l2", "l3", 3, 4.5, tmp{6})
	AsyncAlert(logrus.WarnLevel, "调试-警告", "l4", "l5")

	time.Sleep(time.Second)
}
