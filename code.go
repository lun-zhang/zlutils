package zlutils

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
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

func (code Code) Error() string {
	if code.Err != nil {
		return code.Err.Error()
	}
	return fmt.Sprintf("ret: %d, msg: %s", code.Ret, code.Msg)
}

var retMap = map[int]struct{}{}

func CodeAdd(ret int, msg string) (code Code) {
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

func CodeSend(c *gin.Context, data interface{}, err error) {
	var code Code
	if err == nil {
		code = CodeSuccess
	} else {
		var ok bool
		if code, ok = err.(Code); !ok {
			code = CodeServerErr.WithError(err) //NOTE: 未定义的会被认为是服务器错误，因此客户端错误一定都要定义
		}
	}

	if logrus.IsLevelEnabled(logrus.DebugLevel) && code.Err != nil { //NOTE: err在debug模式拼接到msg上，正式环境不会输出
		code.Msg = fmt.Sprintf("%s: %s", code.Msg, code.Err.Error())
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
	CodeSuccess = CodeAdd(0, "success")
	//服务器错误，msg统一为server error，避免泄露服务器信息
	CodeServerErr         = CodeAdd(5000, "server error")
	CodeServerMidPaincErr = CodeAdd(5201, "server error") //panic，被recover了
	CodeServerMidRedisErr = CodeAdd(5202, "server error") //redis错误
	CodeServerRpcErr      = CodeAdd(5100, "server error") //调用其他服务错误，可能是本服务传参错误，也可能是远程服务器错误
	//客户端错误
	CodeClientRequestErr          = CodeAdd(4000, "request failed")
	CodeClientQueryParamsErr      = CodeAdd(4002, "verify query params failed")
	CodeClientPostParamsErr       = CodeAdd(4004, "verify post params failed")
	CodeClientHeaderParamsErr     = CodeAdd(4005, "verify header params failed")
	CodeClientForbidConcurrentErr = CodeAdd(4201, "forbid concurrent by same user")
)
