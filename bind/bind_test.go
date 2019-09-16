package bind

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
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
