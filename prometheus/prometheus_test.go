package prometheus

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"net/http"
	"testing"
	"time"
	"zlutils/logger"
	"zlutils/request"
)

func TestGetAddress(t *testing.T) {
	fmt.Println(GetAddress())
}

func TestRegister(t *testing.T) {
	logger.Init(logger.Config{Level: logrus.DebugLevel})
	GetAddress = func() (address string, err error) {
		time.Sleep(time.Second)
		return "127.0.0.1:9998", nil
	}
	go func() {
		err := Register(request.Config{
			Method: http.MethodPost,
			Url:    "http://test-m.videobuddy.vid007.com/api/operations_rpc/consul/prometheus/job_modify",
		}, "counter", "/counter/metrics")
		if err != nil {
			t.Fatal(err)
		}
	}()
	defer Unregister()
	time.Sleep(time.Second * 2)
}
