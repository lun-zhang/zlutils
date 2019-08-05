package request

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"net/http"
	"testing"
	"time"
	"zlutils/logger"
	zt "zlutils/time"
)

var ctx context.Context

func init() {
	ctx = context.Background()
	//ctx, _ = xray.BeginSegment(context.Background(), "init")
	logger.Init(logger.Config{Level: logrus.DebugLevel})
}

var (
	addConfig = Config{
		Method: http.MethodPost,
		Url:    "http://localhost:9998/counter/add",
		Client: &ClientConfig{
			Timeout: zt.Duration{
				Duration: time.Second,
			},
		},
	}

	listConfig = Config{
		Method: http.MethodGet,
		Url:    "http://localhost:9998/counter/list?behavior_type=like&is=-3",
	}
)

func TestReqPost(t *testing.T) {
	req := Request{
		Config: addConfig,
		Body: MSI{
			"like": []int64{1},
		},
		Header: MSI{
			"Product-Id": 45,
			"User-Id":    "1",
		},
	}
	var respBody RespRet
	if err := req.Do(ctx, &respBody); err != nil {
		t.Fatal(err)
	}
	fmt.Println(respBody)
}

func TestReqGet(t *testing.T) {
	req := Request{
		Config: listConfig,
		Body: MSI{
			"pub_id": 1,
		},
		Query: MSI{
			"is":   -2,
			"size": 100,
		},
		Header: MSI{
			"Product-Id": 45,
			"User-Id":    "1",
		},
	}
	for i := 0; i < 3; i++ {
		req.AddQuery("is", i) //用于数组
	}

	var respBody struct {
		RespRet
		Data interface{} `json:"data"`
	}

	if err := req.Do(ctx, &respBody); err != nil {
		fmt.Printf("%#v\n", err)
		t.Fatal(err)
	}
	fmt.Printf("%+v\n", respBody)
}
