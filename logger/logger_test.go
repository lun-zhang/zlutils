package logger

import (
	"context"
	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	"testing"
	"time"
	"xlbj-gitlab.xunlei.cn/oversea/zlutils/v7/guard"
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
	router.Group("", MidDebug()).POST("logger", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"a": gin.H{
				"b": "s",
				"i": 1,
			},
		})
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

//验证：
//1.同一个线程的多个日志的trace_id相同
//2.不同线程之间trace_id不同
func TestTraceId(t *testing.T) {
	Init(Config{Level: logrus.DebugLevel})
	for i := 1; i <= 2; i++ {
		go func(id int) {
			ctx, _ := xray.BeginSegment(context.Background(), "test")
			f(ctx, id, 1)
		}(i)
	}
	select {}
}

func f(ctx context.Context, id int, dep int) (err error) {
	defer guard.BeforeCtx(&ctx)(&err)
	if dep > 2 {
		return nil
	}
	entry := logrus.WithContext(ctx)
	entry.WithFields(logrus.Fields{
		"id":  id,
		"dep": dep,
	}).Info()
	return f(ctx, id, dep+1)
}
