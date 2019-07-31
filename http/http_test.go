package http

import (
	"context"
	"fmt"
	"github.com/aws/aws-xray-sdk-go/xray"
	"net/http"
	"testing"
	"time"
	"zlutils/consul"
)

var ctx context.Context

func init() {
	ctx, _ = xray.BeginSegment(context.Background(), "init")
}

var (
	addConfig = RequestConfig{
		Method: http.MethodPost,
		Url:    "http://localhost:9998/counter/add",
		ClientConfig: &ClientConfig{
			Timeout: consul.Duration{
				Duration: time.Second,
			},
		},
	}

	listConfig = RequestConfig{
		Method: http.MethodGet,
		Url:    "http://localhost:9998/counter/list?behavior_type=like&is=-3",
	}
)

func TestReqPost(t *testing.T) {
	req := Request{
		RequestConfig: addConfig,
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
		RequestConfig: listConfig,
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
