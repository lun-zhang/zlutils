package request

import (
	"context"
	"fmt"
	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator"
	"github.com/sirupsen/logrus"
	"net/http"
	"testing"
	"time"
	"zlutils/code"
	"zlutils/guard"
	"zlutils/logger"
	"zlutils/metric"
	zt "zlutils/time"
	zx "zlutils/xray"
)

var ctx context.Context

func init() {
	ctx = context.Background()
	var seg *xray.Segment
	ctx, seg = xray.BeginSegment(context.Background(), "init")
	guard.DoBeforeCtx, guard.DoAfter = zx.DoBeforeCtx, zx.DoAfter
	_ = seg
	logger.Init(logger.Config{Level: logrus.DebugLevel})
}

func f(ctx context.Context) (err error) {
	defer guard.BeforeCtx(&ctx)(&err)

	return f2(ctx)
}
func f2(ctx context.Context) (err error) {
	defer guard.BeforeCtx(&ctx)(&err)

	seg := xray.GetSegment(ctx)
	_ = seg
	config := Config{
		Method: http.MethodPost,
		Url:    "http://localhost:11151/info/4",
		Client: &ClientConfig{
			Timeout: zt.Duration{Duration: time.Second * 2},
		},
	}
	req := Request{
		Config: config,
		Query: MSI{
			"q": 1,
		},
		Header: MSI{
			"H": 2,
		},
		Body: MSI{
			"b": 3,
		},
	}
	var resp struct {
		RetMsg
		Data struct {
			R int `json:"r"`
		}
	}
	if err = req.Do(ctx, &resp); err != nil {
		return
	}
	return
}

func TestInfo(t *testing.T) {
	f(ctx)
}

type RetMsg struct {
	Ret int    `json:"ret"`
	Msg string `json:"msg"`
}

func (m RetMsg) Check() error {
	if m.Ret != 0 {
		return fmt.Errorf("ret:%d msg:%s", m.Ret, m.Msg)
	}
	return nil
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
		RetMsg
		Data interface{} `json:"data"`
	}

	if err := req.Do(ctx, &respBody); err != nil {
		fmt.Printf("%#v\n", err)
		t.Fatal(err)
	}
	fmt.Printf("%+v\n", respBody)
}

func TestVali(t *testing.T) {
	fmt.Println("1", validator.New().Struct(Config{}))
	fmt.Println("2", validator.New().Struct(Config{
		Method: "get",
		//Url:"http://a.com",
	}))
	fmt.Println("3", validator.New().Struct(Config{
		Method: "get",
		Url:    "http://a.com",
	}))
	fmt.Println("4", validator.New().Struct(Config{
		Method: "GET",
	}))
	fmt.Println("5", validator.New().Struct(Config{
		Method: "GET",
		Url:    "abc",
	}))
	fmt.Println("6", validator.New().Struct(Config{
		Method: "GET",
		Url:    "http://a.com?a=1&b=c=a",
	}))
}
func TestPass(t *testing.T) {
	code.SetCodePrefix(3)
	clientErr1 := code.AddLocal(-1, "this client error")
	router := gin.New()
	router.Use(code.MidRespCounterErr("rpc"))
	router.GET("request/metrics", metric.Metrics)
	router.GET("request/pass", code.MidRespWithErr(false),
		func(c *gin.Context) {
			req := Request{
				Config: Config{
					Url:    "http://localhost:12345/code/multi",
					Method: http.MethodGet,
				},
			}
			var resp struct {
				RespPass
			}
			err := req.Do(ctx, &resp)
			code.Send(c, 1, err)
		})
	router.GET("request/client_err", func(c *gin.Context) {
		code.Send(c, 2, clientErr1)
	})
	router.Run(":12346")
}

func TestTryGetItemsIfSlice(t *testing.T) {
	a := []int{1, 2, 3}
	var b [3]int
	b[0] = 10
	for i, test := range []struct {
		in interface{}
		ok bool
	}{
		{a, true},
		{b, true},
		{3, false},
	} {
		s, ok := tryGetItemsIfSlice(test.in)
		if ok != test.ok {
			t.Errorf("%d faild, get %v want %v,in:%v", i, ok, test.ok, test.in)
		} else {
			t.Logf("%d pass, in:%v, s:%v", i, test.in, s)
		}
	}
}

func TestQuerySlice(t *testing.T) {
	a := []int{1, 2, 3}
	var b [3]int
	b[0] = 10
	req := Request{
		Config: Config{
			Url: "http:/a.com",
		},
		Query: MSI{
			"a": a,
			"b": b,
			"c": []string{"A", "B"},
			"d": 1.2,
		},
	}
	s, err := req.GetUrl(ctx)
	if err != nil {
		t.Error(err)
	} else {
		fmt.Println(s)
	}
}
