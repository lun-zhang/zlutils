package logger

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	"testing"
	"time"
	"zlutils/caller"
	"zlutils/metric"
)

func TestLog(t *testing.T) {
	caller.Init("zlutils")
	Init(Config{
		Level: logrus.DebugLevel,
	})
	logrus.Debugf("d")
	logrus.Infof("i")
	logrus.Error("e")
	logrus.Warnf("w")
}

func TestMidDebug(t *testing.T) {
	caller.Init("zlutils")
	Init(Config{Level: logrus.DebugLevel})
	router := gin.New()
	router.Group("", MidDebug()).GET("logger", func(c *gin.Context) {
		c.JSON(http.StatusOK, "resp data")
		logrus.SetLevel(logrus.InfoLevel) //下次调用就不会输出debug日志
	})
	router.Run(":11114")
}

func TestMetric(t *testing.T) {
	const projectName = "zlutils"
	Init(Config{Level: logrus.InfoLevel})
	InitDefaultMetric(projectName) //这一行注释掉后，metric就没有log count了
	router := gin.New()
	router.Group(projectName).GET("metrics", metric.Metrics)
	go func() {
		for {
			logrus.Info("i")
			logrus.Error("e")
			time.Sleep(time.Second)
		}
	}()
	router.Run(":11118")
}

func TestMidInfo(t *testing.T) {
	router := gin.New()
	router.Use(MidInfo())
	router.GET("info", func(c *gin.Context) {
		time.Sleep(time.Millisecond * 100)
		c.JSON(http.StatusOK, 1)
	})
	router.Run(":11121")
}
