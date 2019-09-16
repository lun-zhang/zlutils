package mysql

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"testing"
	"time"
	"zlutils/caller"
	"zlutils/logger"
	"zlutils/metric"
)

func TestGorm(t *testing.T) {
	caller.Init("zlutils")
	logger.Init(logger.Config{Level: logrus.DebugLevel})
	db := New(Config{
		Url: "root:123@/counter?charset=utf8&parseTime=True&loc=Local",
	})
	var cs Counter
	if err := db.Find(&cs).Error; err != nil {
		t.Fatal(err)
	}
	fmt.Println(cs)
}

type Counter struct {
	BehaviorType string `gorm:"column:behavior_type"`
	PubId        int64  `gorm:"column:pub_id"`
	Count        int64  `gorm:"column:count"`
}

func (Counter) TableName() string {
	return "counter"
}

func TestMetric(t *testing.T) {
	const projectName = "zlutils"
	InitDefaultMetric(projectName)
	router := gin.New()
	router.Group(projectName).GET("metrics", metric.Metrics)
	go func() {
		for {
			TestGorm(t)
			time.Sleep(time.Second)
		}
	}()
	router.Run(":11119")
}
