package consul

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"testing"
	"zlutils/logger"
	"zlutils/request"
	"zlutils/time"
)

type Tmp struct {
	D *time.Duration `json:"d" validate:"required"`
}

func TestWatchJson(t *testing.T) {
	Init(":8500", "test/service/counter")
	var tmp Tmp
	ValiStruct().WatchJson("tmp", &tmp, func() {
		//panic(1)
		fmt.Println("change to", tmp)
	})
	fmt.Println(tmp) //{0s}
	select {}
}

func TestGetJson(t *testing.T) {
	Init(":8500", "test/service/counter")
	var tmp Tmp
	GetJson("tmp", &tmp)
	fmt.Println(tmp)
}

func TestBindRouter(t *testing.T) {
	Init(":8500", "test/service/counter")
	var a struct {
		I int `json:"i" validate:"min=2" binding:"min=2"`
	}
	var log logger.Config
	GetJson("a", &a)
	GetJson("log", &log)
	var m map[string]struct{}
	GetJson("m", &m)

	r := gin.New()
	admin := r.Group("admin")

	BindRouter(admin)
	r.Run(":11130")
}

func TestGetJsonValiStruct(t *testing.T) {
	Init(":8500", "test/service/counter")
	var yoyo request.Config
	ValiStruct().GetJson("yoyo", &yoyo)
	var b int
	ValiVar("min=2").GetJson("b", &b)
}

func TestWatchJsonHandler(t *testing.T) {
	Init(":8500", "test/service/counter")
	ValiStruct().WatchJsonVarious("tmp", func(tmp Tmp) {
		fmt.Println(tmp)
	})
	select {}
}

func TestGetJsonHandler(t *testing.T) {
	Init(":8500", "test/service/counter")
	GetJson("log_watch", func(log logger.Config) {
		fmt.Println(log.Level)
	})
}
