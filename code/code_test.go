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

	endless.ListenAndServe(":11111", router)
	os.Exit(0)
}
