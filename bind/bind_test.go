package bind

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	"reflect"
	"testing"
	"zlutils/caller"
	"zlutils/code"
	"zlutils/guard"
	"zlutils/logger"
	"zlutils/meta"
	"zlutils/xray"
)

func init() {
	caller.Init("zlutils")
	logger.Init(logger.Config{Level: logrus.DebugLevel})
}

func TestWrap(t *testing.T) {
	router := gin.New()
	base := router.Group("", code.MidRespWithErr(false))
	base.Group("", func(c *gin.Context) {
		c.Set("a", 1)
	}).POST("api/:u", Wrap(api))
	base.GET("no_req", Wrap(noReq))
	base.GET("no_resp", Wrap(noResp))
	base.GET("resp", Wrap(resp))
	base.GET("resp2", Wrap(resp2))
	base.GET("err", Wrap(err))
	base.GET("err2", Wrap(err2))
	router.Run(":11150")
}

type Resp struct {
	R interface{} `json:"r"`
}

func api(ctx context.Context, req struct {
	ComBody
	//B2//Body出现多次会被检查出来
	//Body2 int//未识别的名字会被检查出来
	Uri struct {
		U int `uri:"u" binding:"required"`
	}
	Query struct {
		Q int `form:"q" binding:"required"`
	}
	Header struct {
		H int `header:"h" binding:"required"`
	}
	Meta meta.Meta //ok
	//Meta map[string]interface{}//ok
	//Meta int //不是map
	//Meta map[int]interface{}//key不是string
	//Meta map[string]int//value不是interface{}
	C *gin.Context `json:"-"` //NOTE: json.Unmarshal会失败，所以禁掉，避免日志打印时候失败
}) (resp Resp, err error) {
	//entry := logrus.WithField("req", req)
	//entry.Info()
	resp.R = req
	//fmt.Println(req.C.ClientIP())
	//resp=nil
	return
	//return resp,fmt.Errorf("e")
}

func noReq(ctx context.Context) (resp interface{}, err error) {
	return nil, nil
}

func Info(ctx context.Context, req struct {
	Body struct {
		B int `json:"b" binding:"required,oneof=1 2 3"`
	}
	Uri struct {
		U int `uri:"u" binding:"required"`
	}
	Query struct {
		Q int `form:"q" binding:"required"`
	}
	Header struct {
		H int `header:"h" binding:"required"`
	}
	C *gin.Context `json:"-"`
}) (resp struct {
	R int `json:"r"`
}, err error) {
	defer guard.BeforeCtx(&ctx)(&err)
	resp.R = req.Body.B + req.Uri.U + req.Query.Q + req.Header.H
	fmt.Println(req.C.Request.Header)
	return
}

func TestInfo(t *testing.T) {
	router := gin.New()
	router.Use(xray.Mid("zlutils", nil, nil, nil))
	router.POST("info/:u", Wrap(Info))
	router.Run(":11151")
}

func noResp(ctx context.Context)                   {}
func resp(ctx context.Context) (resp interface{})  { return nil }
func resp2(ctx context.Context) (resp interface{}) { return 1 }
func err(ctx context.Context) (err error)          { return nil }
func err2(ctx context.Context) (err error)         { return fmt.Errorf("eee") }

func TestWrapApiErr0(t *testing.T) {
	Wrap(func() {})
}
func TestWrapApiErr1(t *testing.T) {
	Wrap(func(req int) (interface{}, error) { return nil, nil })
}
func TestWrapApiErr2(t *testing.T) {
	Wrap(func(req, req2 int) (interface{}, error) { return nil, nil })
}

func TestWrapApiErr3(t *testing.T) {
	Wrap(func() (interface{}, int) { return nil, 1 })
}

func TestWrapApiErr4(t *testing.T) {
	Wrap(1)
}

type ComBody struct {
	Body string `bind:"reuse_body,ignore_error"`
}
type B2 struct {
	Body int
}

func TestA(t *testing.T) {
	var req struct {
		ComBody
	}
	req.Body = "abc"
	//t:=reflect.TypeOf(req)
	v := reflect.ValueOf(req)
	b := v.FieldByName("Body")
	b.Set(reflect.ValueOf("def"))
	fmt.Println(b)
	//checkReqType(reflect.TypeOf(req))
}

func TestWithSender(t *testing.T) {
	router := gin.New()
	router.POST("sender", WithSender(
		func(c *gin.Context, reqFieldName string, bindErr error) {
			c.JSON(http.StatusBadRequest, gin.H{
				"field":  reqFieldName,
				"result": bindErr.Error(),
			})
		},
		func(c *gin.Context, code int, obj interface{}) {
			c.JSON(code, obj)
		}).Wrap(UseMySender))
	router.Run(":11152")
}

func UseMySender(ctx context.Context, req struct {
	Query struct {
		Q int `form:"q" binding:"required"` //请求的query结构
	}
	Body struct {
		B int `json:"b" binding:"required"` //请求的body结构
	}
}) (code int, obj interface{}) { //code是http状态码，obj会被转成成json作为响应
	switch req.Body.B {
	case 1:
		return http.StatusInternalServerError, gin.H{
			"ret": 5,
			"msg": "server error when use sender",
		}
	case 2:
		return http.StatusOK, gin.H{
			"ret":  0,
			"msg":  "success when use sender",
			"data": 123,
			"tile": 456, //随意定义
		}
	default:
		return http.StatusOK, 1
	}
}
