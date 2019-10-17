package consul

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"testing"
	"xlbj-gitlab.xunlei.cn/oversea/zlutils/v7/request"
	"zlutils/logger"
	"zlutils/time"
)

type Tmp struct {
	D time.Duration `json:"d"`
}

func TestWatchJson(t *testing.T) {
	Init(":8500", "test/service/counter")
	var tmp Tmp
	WatchJson("tmp", &tmp, func() {
		fmt.Println("change to", tmp)
	})
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
	GetJsonValiStruct("yoyo", &yoyo)
	var b int
	GetJsonValiVar("b", &b, "min=2")
}
