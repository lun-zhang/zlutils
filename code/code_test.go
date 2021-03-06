package code

import (
	"encoding/json"
	"fmt"
	"github.com/fvbock/endless"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"os"
	"testing"
	"zlutils/logger"
	"zlutils/misc"
	"zlutils/xray"
)

func TestT(t *testing.T) {
	router := gin.New()
	//router.Use(zlutils.Recovery(),gin.Recovery())

	router.GET("code/ok", func(c *gin.Context) {
		Send(c, "is ok", nil)
	})
	router.GET("code/err", func(c *gin.Context) {
		Send(c, "is err", fmt.Errorf("err info"))
	})
	router.GET("code/err/404", func(c *gin.Context) {
		Send(c, "is err", ClientErr404.WithErrorf("err info"))
	})
	gin.Mode()

	endless.ListenAndServe(":11111", router)
	os.Exit(0)
}

func TestRespIsErr(t *testing.T) {
	c := &gin.Context{}
	fmt.Println(RespIsClientErr(c))
	fmt.Println(RespIsServerErr(c))

	c.Set(keyRet, ClientErr.Ret)
	fmt.Println(RespIsClientErr(c))
	fmt.Println(RespIsServerErr(c))

	c.Set(keyRet, ServerErr.Ret)
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

func TestMidRespWithErr(t *testing.T) {
	//gin.SetMode(gin.ReleaseMode) //这一行注释掉后，app会带上err信息
	router := gin.New()
	router.GET("no", e) //默认任何时候都不显示err信息
	router.Group("app", MidRespWithErr(true)).GET("", e)
	router.Group("rpc", MidRespWithErr(false)).GET("", e)
	router.Run(":11123")
}

func TestMidRespWithTraceId(t *testing.T) {
	gin.SetMode(gin.ReleaseMode) //这一行注释掉后，app会带上trace_id
	router := gin.New()
	router.Use(xray.Mid("zlutils", nil, nil, nil))
	router.GET("no", info) //默认任何时候都不显示trace_id
	router.Group("app", MidRespWithTraceId(true)).GET("", info)
	router.Group("rpc", MidRespWithTraceId(false)).GET("", info)
	router.Run(":11124")
}
func info(c *gin.Context) { Send(c, 1, nil) }

func pj(i interface{}) {
	b, _ := json.Marshal(i)
	fmt.Println(string(b))
}

func TestAdd(t *testing.T) {
	c1 := Add(1, MSS{
		"en": "e",
		"zh": "中",
	})
	pj(c1.cloneByLang("en"))
	pj(c1.cloneByLang("zh"))
	pj(c1.cloneByLang("hi"))
}

func TestAddNoEn(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Log(r)
		} else {
			t.Error("must panic")
		}
	}()
	Add(1, MSS{ //panic
		"zh": "中",
	})
}

func TestMultiLang(t *testing.T) {
	logger.Init(logger.Config{Level: logrus.DebugLevel})
	co := Add(1, MSS{
		"en": "e",
		"zh": "中",
	})
	r := gin.New()
	r.GET("code/multi",
		MidRespWithErr(false),
		logger.MidDebug(),
		func(c *gin.Context) {
			Send(c, nil, co.WithErrorf("with"))
		})
	r.Run(":12345")
}

func TestAddIsClone(t *testing.T) {
	msg := MSS{
		misc.LangEnglish: "e",
	}
	co := Add(1, msg)
	fmt.Println(co.msgMap)
	msg[misc.LangEnglish] = "e2"
	fmt.Println(co.msgMap)
	if co.msgMap[misc.LangEnglish] != "e" {
		t.Error("不能被改变")
	}
}
