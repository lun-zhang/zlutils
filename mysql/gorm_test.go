package mysql

import (
	"context"
	"fmt"
	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/gin-gonic/gin"
	"github.com/lun-zhang/gorm"
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
	db := NewMasterAndSlave(ConfigMasterAndSlave{
		Master: Config{
			Url:          "root:123@/counter?charset=utf8&parseTime=True&loc=Local",
			MaxOpenConns: 4,
		},
		Slave: Config{
			Url:          "root:123@/counter?charset=utf8&parseTime=True&loc=Local",
			MaxOpenConns: 5,
		},
	})
	var cs []Counter
	if err := db.Master().Find(&cs).Error; err != nil {
		t.Fatal(err)
	}
	fmt.Println(cs)
	if len(cs) > 0 {
		fmt.Println(cs[0])
	}
}

type Counter struct {
	Id           int64  `gorm:"column:id;primary_key"`
	BehaviorType string `gorm:"column:behavior_type"`
	An
	unexported string `gorm:"column:unexported"`
}

type An struct {
	PubId int64 `gorm:"column:pub_id"`
	Count int64 `gorm:"column:count"`
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

var ctx, _ = xray.BeginSegment(context.Background(), "test")

func init() {
	logger.Init(logger.Config{Level: logrus.DebugLevel})
}

func TestJustLogger(t *testing.T) {
	const projectName = "zlutils"
	InitDefaultMetric(projectName)
	router := gin.New()
	router.Group(projectName).GET("metrics", metric.Metrics)
	go func() {
		//这里的gorm可以是jinzhu的
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

func TestMetricLogger_Print(t *testing.T) {
	const projectName = "zlutils"
	InitDefaultMetric(projectName)
	router := gin.New()
	router.Group(projectName).GET("metrics", metric.Metrics)
	go func() {
		dbConn := New(Config{Url: "root:123@/counter?charset=utf8&parseTime=True&loc=Local"})
		for {
			var cs []Counter
			if err := dbConn.
				WithContext(ctx).
				Find(&cs).
				Error; err != nil {
				panic(err)
			}
			fmt.Println(len(cs))
			time.Sleep(time.Second)
		}
	}()
	router.Run(":11119")
}

func TestOmitColumns(t *testing.T) {
	c := Counter{
		Id:           1,
		BehaviorType: "a",
		An: An{
			PubId: 2,
			Count: 3,
		},
		unexported: "un",
	}

	for i, test := range []struct {
		omits []string
	}{
		{[]string{}},
		{[]string{"id"}},
		{[]string{"behavior_type"}},
		{[]string{"pub_id"}},
		{[]string{"behavior_type"}},
		{[]string{"id", "behavior_type"}},
		{[]string{"id", "behavior_type", "pub_id"}},
		{[]string{"id", "behavior_type", "pub_id", "count"}},
	} {
		fmt.Println(i, test.omits, OmitColumns(c, test.omits...))
	}
}
