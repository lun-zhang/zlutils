package db

import (
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"
	"testing"
	"zlutils/caller"
	"zlutils/log"
)

func TestGorm(t *testing.T) {
	caller.Init("zlutils")
	log.Init(log.Config{Level: logrus.DebugLevel})
	db := NewDB(Config{
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