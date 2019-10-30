package code

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	"reflect"
	"zlutils/misc"
)

type Code struct {
	Ret    int    `json:"ret"`
	Msg    string `json:"msg"`
	msgMap MLS    `json:"-"` //多语言的msg
	Err    error  `json:"-"` //真实的err，用于debug返回
}

//复制一份，否则线程竞争
func (code Code) cloneByLang(lang string) Code {
	if msg, ok := code.msgMap[lang]; ok {
		code.Msg = msg
		return code
	}
	//没有则取英语的
	if msg, ok := code.msgMap[LangEn]; ok {
		code.Msg = msg
		return code
	}
	//通过Add和AddMultiLang生成的Code一定不会到这里
	//只有可能是故意让ret重复时，直接创建的Code对象
	return code
}

func (code Code) WithError(err error) Code {
	code.Err = err
	return code
}
func (code Code) WithErrorf(format string, a ...interface{}) Code {
	return code.WithError(fmt.Errorf(format, a...))
}

func (code Code) Error() string {
	if code.Err != nil {
		return code.Err.Error()
	}
	msg := code.Msg
	if msg == "" {
		msg = code.msgMap[LangEn]
	}
	return fmt.Sprintf("ret: %d, msg: %s", code.Ret, msg)
}

var retMap = map[int]struct{}{}

const (
	LangEn = "en"
	LangHi = "hi-IN"
)

type MLS map[string]string

//如果msg是string，则当做英语
//如果msg是map，那么会复制一份，所以可以放心不会被修改
//如果msg是其他类型则panic
func Add(ret int, msg interface{}) (code Code) {
	var msgMap MLS
	switch msg := msg.(type) {
	case string:
		msgMap = MLS{
			LangEn: msg, //默认认为是英语
		}
	case MLS:
		msgMap = MLS{}
		for k, v := range msg {
			msgMap[k] = v //复制一份避免被修改
		}
	default:
		logrus.Panicf("invalid msg type:%s", reflect.TypeOf(msg))
	}
	return add(ret, msgMap)
}

func add(ret int, msgMap MLS) (code Code) {
	if _, ok := retMap[ret]; ok {
		panic(fmt.Errorf("ret %d exist", ret)) //NOTE: 禁止传相同的ret
	}
	retMap[ret] = struct{}{}
	if _, ok := msgMap[LangEn]; !ok {
		panic(fmt.Errorf("no english msg")) //必须有英语的msg，空的也允许
	}
	code = Code{
		Ret:    ret,
		msgMap: msgMap,
	}
	return code
}

type Result struct {
	Code
	Data interface{} `json:"data,omitempty"`
}

const KeyRet = "_key_ret"

func IsServerErr(ret int) bool {
	return ret >= 5000 && ret < 6000
}
func IsClientErr(ret int) bool {
	return ret >= 4000 && ret < 5000
}

type Context struct {
	*gin.Context
}

type HandelFunc func(*Context)

func Wrap(f HandelFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		f(&Context{c})
	}
}

func WrapSend(f func(c *gin.Context) (resp interface{}, err error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp, err := f(c)
		Send(c, resp, err)
	}
}

const keyRespWithErr = "_key_resp_show_err"

//使用此中间件的接口，输出带上err信息
//closeInRelease=true时候，则不在正式环境输出，其他环境输出
//例如app接口，正式环境不输出，测试环境输出，则设置closeInRelease=true
//admin、rpc接口任何环境都输出，则设置closeInRelease=false
func MidRespWithErr(closeInRelease bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if closeInRelease && gin.Mode() == gin.ReleaseMode {
			return
		}
		c.Set(keyRespWithErr, 1) //数字没啥意义
	}
}

func GetRet(c *gin.Context) (int, bool) {
	if v, ok := c.Get(KeyRet); ok {
		if ret, ok := v.(int); ok {
			return ret, ok
		}
	}
	return 0, false
}

func RespIsServerErr(c *gin.Context) bool {
	if ret, ok := GetRet(c); ok {
		return IsServerErr(ret)
	}
	return false
}

func RespIsClientErr(c *gin.Context) bool {
	if ret, ok := GetRet(c); ok {
		return IsClientErr(ret)
	}
	return false
}

func (c *Context) Send(data interface{}, err error) {
	Send(c.Context, data, err)
}

var Send = func(c *gin.Context, data interface{}, err error) {
	var code Code
	if err == nil {
		code = Success
	} else {
		var ok bool
		if code, ok = err.(Code); !ok {
			code = ServerErr.WithError(err) //NOTE: 未定义的会被认为是服务器错误，因此客户端错误一定都要定义
		}
	}
	reqHeader := c.Request.Header
	lang := reqHeader.Get("Device-Language") //优先取站内
	if lang == "" {
		lang = reqHeader.Get("Accept-Language") //没有则可能在站外
	}
	code = code.cloneByLang(lang) //复制，避免线程竞争

	if code.Err != nil {
		if _, ok := c.Get(keyRespWithErr); ok {
			if code.Msg == "" {
				code.Msg = code.Err.Error()
			} else {
				code.Msg = fmt.Sprintf("%s: %s", code.Msg, code.Err.Error())
			}
		}
	}

	if code.Ret != 0 || //不是成功就不反回data
		misc.IsNil(data) { //如果data设为nil则也不返回
		data = nil
	}
	c.Set(KeyRet, code.Ret) //保存ret用于metrics
	c.JSON(http.StatusOK, Result{
		Code: code,
		Data: data,
	})
}

/*
ret统一，方便prometheus统计
正确:			0
客户端参数错误:	40xx
客户端逻辑错误:	41xx
客户端其他错误:	42xx
服务器错误:		50xx
服务器rcp错误:	5100
服务器中间件错误	52xx
*/

var (
	//成功
	Success = Add(0, "success")
	//服务器错误，msg统一为server error，避免泄露服务器信息
	ServerErr      = Add(5000, "server error")
	ServerErrPainc = Add(5201, "server error") //panic，被recover了
	ServerErrRedis = Add(5202, "server error") //redis错误
	ServerErrRpc   = Add(5100, "server error") //调用其他服务错误，可能是本服务传参错误，也可能是远程服务器错误
	//客户端错误
	ClientErr                 = Add(4000, "client error")
	ClientErrQuery            = Add(4002, "verify query params failed")
	ClientErrBody             = Add(4004, "verify body params failed")
	ClientErrHeader           = Add(4005, "verify header params failed")
	ClientErrUri              = Add(4006, "verify uri params failed")
	ClientErr404              = Add(4040, "not found")
	ClientErrForbidConcurrent = Add(4201, "forbid concurrent by same user")
)
