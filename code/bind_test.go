package code

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"testing"
	"zlutils/caller"
	"zlutils/logger"
)

func init() {
	caller.Init("zlutils")
	logger.Init(logger.Config{Level: logrus.DebugLevel})
}

func TestWrapApi(t *testing.T) {
	router := gin.New()
	base := router.Group("", MidRespWithErr(false))
	base.POST("api/:u", WrapApi(api))
	base.GET("no_req", WrapApi(noReq))
	router.Run(":11150")
}

func api(req struct {
	Body struct {
		B int `json:"b"`
	}
	Uri struct {
		U int `uri:"u"`
	}
	Query struct {
		Q int `form:"q"`
	}
}) (resp struct {
	R interface{} `json:"r"`
}, err error) {
	resp.R = req
	return
	//return resp,fmt.Errorf("e")
}

func noReq() (resp interface{}, err error) {
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
