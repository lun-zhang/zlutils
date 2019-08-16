package session

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	"testing"
	"zlutils/code"
	"zlutils/logger"
	"zlutils/request"
)

func TestMidUser(t *testing.T) {
	router := gin.New()
	router.Group("user/default", MidUser(nil)).GET("", u)
	router.Group("user/code", MidUser(code.SendClientErrHeader)).GET("", u)
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
	router.Group("operator/default", MidOperator(nil)).GET("", o)
	router.Group("operator/code", MidOperator(code.SendClientErrQuery)).GET("", o)
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
	router.Group("user/no_mid_user", MidBindUserVideoBuddy(bind, code.SendServerErrRpc)).GET("", u)
	hasUser := router.Group("user/has", MidUser(nil))
	hasUser.Group("default", MidBindUserVideoBuddy(bind, nil)).GET("", u)
	hasUser.Group("code",
		code.MidRespWithErr(true),
		MidBindUserVideoBuddy(bind, code.SendServerErrRpc)).GET("", u)
	router.Run(":11116")
}
