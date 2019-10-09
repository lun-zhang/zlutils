package code

import (
	"encoding/json"
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

func ew(c *gin.Context) (resp interface{}, err error) {
	var reqQuery struct {
		I int `form:"i"`
	}
	if err = c.ShouldBindQuery(&reqQuery); err != nil {
		err = ClientErrQuery.WithError(err)
		return
	}
	return do(reqQuery.I)
}

var clientErrI0 = Add(4101, "i is 0")

func do(i int) (resp struct {
	I int
}, err error) {
	if i == 0 {
		err = clientErrI0.WithErrorf("i=0")
		return
	}
	resp.I = i
	return
}

func TestWrapSend(t *testing.T) {
	router := gin.New()
	router.Group("", MidRespWithErr(false)).GET("we", WrapSend(ew))
	router.Run(":11124")
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

func pj(i interface{}) {
	b, _ := json.Marshal(i)
	fmt.Println(string(b))
}

func TestAddMultiLang(t *testing.T) {
	c1 := AddMultiLang(1, MLS{
		"en": "e",
		"zh": "中",
	})
	pj(c1.cloneByLang("en"))
	pj(c1.cloneByLang("zh"))
	pj(c1.cloneByLang("hi"))
}

func TestAddMultiLangNoEn(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Log(r)
		} else {
			t.Error("must panic")
		}
	}()
	AddMultiLang(1, MLS{ //panic
		"zh": "中",
	})
}

func TestMultiLang(t *testing.T) {
	co := AddMultiLang(1, MLS{
		"en": "e",
		"zh": "中",
	})
	r := gin.New()
	r.Group("", MidRespWithErr(false)).
		GET("code/multi", func(c *gin.Context) {
			Send(c, nil, co.WithErrorf("with"))
		})
	r.Run(":12345")
}
