package consul

import (
	"fmt"
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
