package prometheus

import (
	"fmt"
	"net/http"
	"testing"
	"zlutils/request"
)

func TestGetAddress(t *testing.T) {
	fmt.Println(GetAddress())
}

func TestRegister(t *testing.T) {
	GetAddress = func() (address string, err error) {
		return "127.0.0.1:9998", nil
	}
	un, err := Register(request.Config{
		Method: http.MethodPost,
		Url:    "http://test-m.videobuddy.vid007.com/api/operations_rpc/consul/prometheus/job_modify?caller=counter",
	}, "counter", "/counter/metrics")
	if err != nil {
		t.Fatal(err)
	}
	defer un()
}
