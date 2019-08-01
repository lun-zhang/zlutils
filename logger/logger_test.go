package logger

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	"testing"
	"zlutils/caller"
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
	})
	router.Run(":11114")
}
