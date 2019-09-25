package bind

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"reflect"
	"testing"
	"zlutils/caller"
	"zlutils/code"
	"zlutils/logger"
	"zlutils/meta"
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
}) (resp struct {
	R interface{} `json:"r"`
}, err error) {
	resp.R = req
	return
	//return resp,fmt.Errorf("e")
}

func noReq(ctx context.Context) (resp interface{}, err error) {
	return nil, nil
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
	Body string
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
