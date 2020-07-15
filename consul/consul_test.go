package consul

import (
	"fmt"
	"testing"
	"time"
	"zlutils/logger"
	"zlutils/request"
	zt "zlutils/time"
)

type Tmp struct {
	D *zt.Duration `json:"d" validate:"required"`
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

func TestGetYaml(t *testing.T) {
	Init(":8500", "test/service/counter")
	var ty struct {
		I int               `yaml:"i" validate:"required"`
		S string            `yaml:"s" validate:"required"`
		M map[string]string `yaml:"m" validate:"required"`
		F float64           `yaml:"f" validate:"required"`
	}
	ValiStruct().GetYaml("t.yaml", &ty)
	fmt.Println(ty)
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

func TestWithPrefix(t *testing.T) {
	Init(":8500", "test/service/counter")
	lo := WithPrefix("test/service/example")
	lo.ValiVar("len=15").GetJson("eee", func(re string) {
		fmt.Println(re)
	})
	ValiStruct().GetJson("redis", func(redis struct {
		Url      string        `json:"url" validate:"url"`
		Duration time.Duration `json:"duration"`
	}) {
		fmt.Println(redis)
	})
}
