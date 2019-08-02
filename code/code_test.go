package code

import (
	"fmt"
	"github.com/fvbock/endless"
	"github.com/gin-gonic/gin"
	"os"
	"testing"
)

func TestT(t *testing.T) {
	router := gin.New()
	//router.Use(zlutils.Recovery(),gin.Recovery())

	router.GET("code/ok", Wrap(func(c *Context) {
		c.Send("is ok", nil)
	}))
	router.GET("code/err", Wrap(func(c *Context) {
		c.Send("is err", fmt.Errorf("err info"))
	}))
	router.GET("code/err/404", Wrap(func(c *Context) {
		c.Send("is err", ClientErr404.WithErrorf("err info"))
	}))
	gin.Mode()

	endless.ListenAndServe(":11111", router)
	os.Exit(0)
}

func TestRespIsErr(t *testing.T) {
	c := &gin.Context{}
	fmt.Println(RespIsClientErr(c))
	fmt.Println(RespIsServerErr(c))

	c.Set(KeyRet, ClientErr.Ret)
	fmt.Println(RespIsClientErr(c))
	fmt.Println(RespIsServerErr(c))

	c.Set(KeyRet, ServerErr.Ret)
	fmt.Println(RespIsClientErr(c))
	fmt.Println(RespIsServerErr(c))
}

func e(c *gin.Context) {
	Send(c, 1, fmt.Errorf("e"))
}

func TestMidRespWithErr(t *testing.T) {
	gin.SetMode(gin.ReleaseMode) //这一行注释掉后，app会带上err信息
	router := gin.New()
	router.GET("no", e) //默认任何时候都不显示err信息
	app := router.Group("app", MidRespWithErr(true))
	app.GET("", e)
	rpc := router.Group("rpc", MidRespWithErr(false))
	rpc.GET("", e)
	router.Run(":11123")
}
