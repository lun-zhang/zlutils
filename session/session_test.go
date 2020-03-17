package session

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	"testing"
	"zlutils/bind"
	"zlutils/code"
	"zlutils/logger"
	"zlutils/request"
)

func TestMidUser(t *testing.T) {
	router := gin.New()
	router.Group("user/default", MidUser()).GET("", u)
	router.Group("user/no_vali", WithoutValidate().MidUser()).GET("", u)
	router.Group("user/code", code.MidRespWithErr(true),
		MidUser()).GET("", u)
	router.Group("", code.MidRespWithErr(true),
		MidUser()).GET("user/meta", bind.Wrap(func(ctx context.Context, req struct {
		Header struct {
			UserId   string `header:"User-Id" binding:"required"`
			DeviceId string `header:"Device-Id" binding:"required"`
		}
		Meta Meta
	}) (resp interface{}, err error) {
		resp = req
		fmt.Printf("%+v\n", req.Meta.GetUser())
		fmt.Println(req.Meta.Meta())
		return
	}))
	router.Run(":11115")
}
func u(c *gin.Context) {
	c.JSON(http.StatusOK, GetUser(c))
}
func o(c *gin.Context) {
	c.JSON(http.StatusOK, GetOperator(c))
}

func TestMidOperator(t *testing.T) {
	router := gin.New()
	router.Group("operator/default", MidOperator()).GET("", o)
	router.Group("operator/code", MidOperator()).GET("", o)
	router.Run(":11116")
}

func TestMidBindUserVideoBuddy(t *testing.T) {
	logger.Init(logger.Config{
		Level: logrus.DebugLevel,
	})
	router := gin.New()
	bind := request.Config{
		Method: http.MethodGet,
		Url:    "http://test-m.videobuddy.vid007.com/vcoin_rpc/v1/user_device/binding/get?caller=task_wall",
	}
	router.Group("user/no_mid_user", MidBindUserVideoBuddy(bind)).GET("", u)
	hasUser := router.Group("user/has", MidUser())
	hasUser.Group("default", MidBindUserVideoBuddy(bind)).GET("", u)
	hasUser.Group("code",
		code.MidRespWithErr(true),
		MidBindUserVideoBuddy(bind)).GET("", u)
	router.Run(":11116")
}
