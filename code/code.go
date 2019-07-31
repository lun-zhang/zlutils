package code

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

type Code struct {
	Ret int    `json:"ret"`
	Msg string `json:"msg"`
	Err error  `json:"-"` //真实的err，用于debug返回
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
	return fmt.Sprintf("ret: %d, msg: %s", code.Ret, code.Msg)
}

var retMap = map[int]struct{}{}

func Add(ret int, msg string) (code Code) {
	if _, ok := retMap[ret]; ok {
		panic(fmt.Errorf("ret %d exist", ret)) //NOTE: 禁止传相同的ret
	}
	retMap[ret] = struct{}{}
	code = Code{
		Ret: ret,
		Msg: msg,
	}
	return code
}

type Result struct {
	Code
	Data interface{} `json:"data,omitempty"`
}

const KeyRet = "_key_ret"

func (code Code) IsServerErr(ret int) bool {
	return ret >= 5000 && ret < 6000
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

func (c *Context) Send(data interface{}, err error) {
	var code Code
	if err == nil {
		code = Success
	} else {
		var ok bool
		if code, ok = err.(Code); !ok {
			code = ServerErr.WithError(err) //NOTE: 未定义的会被认为是服务器错误，因此客户端错误一定都要定义
		}
	}
	if code.Err != nil {
		//TODO: 应当只有正式的app不输出，其他都输出，如何解耦？
		//if _, ok := c.Get(KeyAdminOperator); ok || //NOTE: 管理后台请求的错误全部输出，包括服务器错误
		//	logrus.IsLevelEnabled(logrus.DebugLevel) || //NOTE: 非正式环境全部输出，方便调试
		//	!RetIsServerErr(code.Ret) { //NOTE: 正式环境，禁止打印服务器错误，因为可能暴露服务器信息
		if code.Msg == "" {
			code.Msg = code.Err.Error()
		} else {
			code.Msg = fmt.Sprintf("%s: %s", code.Msg, code.Err.Error())
		}
		//}
	}

	if code.Ret != 0 {
		data = nil //NOTE: 不是成功就不反回data
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
