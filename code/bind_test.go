package code

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	"testing"
	"zlutils/caller"
	"zlutils/logger"
	"zlutils/meta"
)

func init() {
	caller.Init("zlutils")
	logger.Init(logger.Config{Level: logrus.DebugLevel})
}

func TestWrapApi(t *testing.T) {
	router := gin.New()
	base := router.Group("", MidRespWithErr(false))
	base.Group("", func(c *gin.Context) {
		c.Set("a", 1)
	}).POST("api/:u", WrapApi(api))
	base.GET("no_req", WrapApi(noReq))
	router.Run(":11150")
}

func api(ctx context.Context, req struct {
	Body struct {
		B int `json:"b" binding:"required"`
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
	return 1, nil
}

func TestWrapApiErr0(t *testing.T) {
	WrapApi(func() {})
}
func TestWrapApiErr1(t *testing.T) {
	WrapApi(func(req int) (interface{}, error) { return nil, nil })
}
func TestWrapApiErr2(t *testing.T) {
	WrapApi(func(req, req2 int) (interface{}, error) { return nil, nil })
}

func TestWrapApiErr3(t *testing.T) {
	WrapApi(func() (interface{}, int) { return nil, 1 })
}

func TestWrapApiErr4(t *testing.T) {
	WrapApi(1)
}

func TestBindHeader(t *testing.T) {
	header := http.Header{}
	header.Add("S", "s")
	header.Add("I", "1")
	header.Add("J", "1")
	header.Add("f", "1.1")
	header.Add("a-b", "1") //被转成大写A-B

	var reqHeader struct {
		S  string  `header:"s"`
		I  int     `header:"-"` //忽略
		J  int     //没tag时候，名字为J
		F  float32 `header:"f"`
		AB int     `header:"A-B"`
		//No int `header:"no" binding:"required"`//检验
	}
	if err := bindHeader(header, &reqHeader); err != nil {
		fmt.Println(err) //如果失败，则reqHeader会被置零
	}
	fmt.Printf("%+v\n", reqHeader)
}
