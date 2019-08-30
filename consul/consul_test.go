package consul

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"testing"
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

func TestGroup(t *testing.T) {
	Init(":8500", "test/service/counter")
	var a struct {
		I int `json:"i" validate:"min=2" binding:"min=2"`
	}
	GetJson("a", &a)

	r := gin.New()
	admin := r.Group("admin")

	InitGroup(admin)
	r.Run(":11130")
}
