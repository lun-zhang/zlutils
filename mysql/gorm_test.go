package mysql

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
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

func TestJustLogger(t *testing.T) {
	const projectName = "zlutils"
	InitDefaultMetric(projectName)
	router := gin.New()
	router.Group(projectName).GET("metrics", metric.Metrics)
	go func() {
		//这里的gorm是jinzhu的
		dbConn, err := gorm.Open("mysql", "root:123@/counter?charset=utf8&parseTime=True&loc=Local")
		if err != nil {
			t.Fatal(err)
		}
		dbConn.LogMode(true).SetLogger(&Logger{})
		for {
			var cs []Counter
			if err := dbConn.Find(&cs).Error; err != nil {
				t.Fatal(err)
			}
			fmt.Println(len(cs))
			time.Sleep(time.Second)
		}
	}()
	router.Run(":11118")
}
