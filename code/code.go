package code

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	"reflect"
	"strings"
	"zlutils/misc"
	"zlutils/xray"
)

//type Interface interface {
//	Code()
//	Message()
//	Error() string
//	WithError(err error) Interface
//	WithErrorf(format string, a ...interface{}) Interface
//}

//默认前缀，不可被任何人显示使用，用于当忘记设置时排查
const defaultCodePrefix = 1000000 //1e6

//错误码前缀，wiki维护，目前是个默认值
var codePrefix = defaultCodePrefix

//服务自己用的错误码上限，因此最大为99999
const localCodeLast = 100000 //1e5

func SetCodePrefix(prefix int) {
	if codePrefix < defaultCodePrefix {
		logrus.Panicf("code prefix:%d must bigger than %d", prefix, defaultCodePrefix)
	}
	codePrefix = prefix
}

/*
rpc拓传错误码和错误信息
目前，我的rpc接口会返回错误的敏感信息，其他人的接口未返回
因此调我rpc接口的服务，不可把敏感信息传出
由于未统一错误码，所以服务之间有相同的错误码，这些错误码不可透传
公共错误码，服务之间不可相同，
*/

type Code struct {
	Ret     int    `json:"ret"`
	Msg     string `json:"msg"`
	TraceId string `json:"trace_id,omitempty"` //跟踪id,用于debug返回，虽然响应的header里有key=x-amzn-trace-id,value="Root=$trace_id"，但是太依赖aws
	//Help    string `json:"help,omitempty"`     //点击此链接调转到错误详情,限制内网

	msgMap MSS    //多语言的msg
	err    error  //真实的err，用于debug返回
	split  string //分割符
	isPass bool   //是否是rpc透传码
}

//是rpc透传码
func (code Code) WithPass() Code {
	code.isPass = true
	return code
}

//复制一份，否则线程竞争
func (code Code) cloneByLang(lang string) Code {
	if msg, ok := code.msgMap[lang]; ok {
		code.Msg = msg
		return code
	}
	//没有则取英语的
	if msg, ok := code.msgMap[misc.LangEnglish]; ok {
		code.Msg = msg
		return code
	}
	//通过Add和AddMultiLang生成的Code一定不会到这里
	//只有可能是故意让ret重复时，直接创建的Code对象
	return code
}

func (code Code) WithError(err error) Code {
	code.err = err
	return code
}
func (code Code) WithErrorf(format string, a ...interface{}) Code {
	return code.WithError(fmt.Errorf(format, a...))
}

var defaultSplit = ": "

func SetDefaultSplit(split string) {
	if split == "" {
		logrus.Panicf("split cant empty")
	}
	defaultSplit = split
}

//NOTE: split不可是msg的子串，否则会在Send函数中被替换成DefaultSplit
func (code Code) WithSplit(split string) Code {
	code.split = split
	return code
}

func (code Code) Error() string {
	if code.err != nil {
		return code.err.Error()
	}
	msg := code.Msg
	if msg == "" {
		msg = code.msgMap[misc.LangEnglish]
	}
	return fmt.Sprintf("ret: %d, msg: %s", code.Ret, msg)
}

var retMap = map[int]struct{}{}

//并没有自己的方法，所以就当个简写
type MSS = map[string]string

//如果msg是string，则当做英语
//如果msg是map，那么会复制一份，所以可以放心不会被修改
//如果msg是其他类型则panic
func Add(retGlobal int, msg interface{}) (code Code) {
	var msgMap MSS
	switch msg := msg.(type) {
	case string:
		msgMap = MSS{
			misc.LangEnglish: msg, //默认认为是英语
		}
	case MSS:
		msgMap = MSS{}
		for k, v := range msg {
			msgMap[k] = v //复制一份避免被修改
		}
	default:
		logrus.Panicf("invalid msg type:%s", reflect.TypeOf(msg))
	}
	return add(retGlobal, msgMap)
}

//由于项目的日志/监控中可能充斥者各个服务的ret，因此必须在Add时候，就把前缀填充
//填充局部码，只能在
func AddLocal(retLocal int, msg interface{}) (code Code) {
	switch {
	case retLocal > 0 && retLocal < localCodeLast:
		retLocal += codePrefix
	case retLocal < 0 && retLocal > -localCodeLast:
		retLocal -= codePrefix
	default:
		logrus.Panicf("invalid retLocal:%d", retLocal)
	}
	return Add(retLocal, msg)
}

func add(ret int, msgMap MSS) (code Code) {
	if _, ok := retMap[ret]; ok {
		panic(fmt.Errorf("ret %d exist", ret)) //NOTE: 禁止传相同的ret
	}
	retMap[ret] = struct{}{}
	if _, ok := msgMap[misc.LangEnglish]; !ok {
		panic(fmt.Errorf("no english msg")) //必须有英语的msg，空的也允许
	}
	code = Code{
		Ret:    ret,
		msgMap: msgMap,
	}
	return code
}

type result struct {
	Code
	Data interface{} `json:"data,omitempty"`
}

const (
	keyRet    = "_key_ret"
	keyIsPass = "_key_is_pass"
)

func isServerErr(ret int) bool {
	//我的旧的
	if ret >= 5000 && ret < 6000 {
		return true
	}
	//统一新的
	if ret > 0 {
		return true
	}

	return false
}
func isClientErr(ret int) bool {
	//我的旧的
	if ret >= 4000 && ret < 5000 {
		return true
	}
	//统一新的
	if ret < 0 {
		return true
	}
	return false
}

const (
	keyRespWithErr     = "_key_resp_show_err"
	keyRespWithTraceId = "_key_resp_trace_id"
)

//使用此中间件的接口，输出带上err信息
//closeInRelease=true时候，则不在正式环境输出，其他环境输出
//例如app接口，正式环境不输出，测试环境输出，则设置closeInRelease=true
//admin、rpc接口任何环境都输出，则设置closeInRelease=false
func MidRespWithErr(closeInRelease bool) gin.HandlerFunc {
	return midMidRespWith(closeInRelease, keyRespWithErr)
}

//使用此中间件，输出带上trace_id
func MidRespWithTraceId(closeInRelease bool) gin.HandlerFunc {
	return midMidRespWith(closeInRelease, keyRespWithTraceId)
}

func midMidRespWith(closeInRelease bool, key string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if closeInRelease && gin.Mode() == gin.ReleaseMode {
			return
		}
		c.Set(key, struct{}{})
	}
}

func getRet(c *gin.Context) (int, bool) {
	if v, ok := c.Get(keyRet); ok {
		if ret, ok := v.(int); ok {
			return ret, ok
		}
	}
	return 0, false
}

func RespIsServerErr(c *gin.Context) bool {
	if statusCode := c.Writer.Status(); statusCode >= 500 && statusCode < 600 {
		return true
	}
	if ret, ok := getRet(c); ok {
		return isServerErr(ret)
	}
	return false
}

func RespIsClientErr(c *gin.Context) bool {
	if statusCode := c.Writer.Status(); statusCode >= 400 && statusCode < 500 {
		return true
	}
	if ret, ok := getRet(c); ok {
		return isClientErr(ret)
	}
	return false
}

func respIsPassErr(c *gin.Context) bool {
	_, ok := c.Get(keyIsPass)
	return ok
}

func Send(c *gin.Context, data interface{}, err error) {
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

	if code.err != nil {
		if _, ok := c.Get(keyRespWithErr); ok {
			//code.Msg如果不是英语，而是用于toast的印地语时，不当输出错误信息
			//用于toast时，就极有可能包含了分隔符
			//当然如果你的接口是用于toast，那么肯定不能把rpc的结果直接返给app
			if code.split == "" || strings.Contains(code.Msg, code.split) {
				code.split = defaultSplit //可以改成". "
			}
			code.Msg = code.Msg + code.split + code.err.Error()
		}
	}
	if _, ok := c.Get(keyRespWithTraceId); ok {
		code.TraceId = xray.GetTraceId(c.Request.Context())
	}

	if code.Ret != 0 || //不是成功就不反回data
		misc.IsNil(data) { //如果data设为nil则也不返回
		data = nil
	}
	if code.isPass {
		c.Set(keyIsPass, struct{}{})
	}
	c.Set(keyRet, code.Ret) //保存ret用于metrics
	c.JSON(http.StatusOK, result{
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
	ServerErrRpc   = Add(5203, "server error") //调用其他服务错误，可能是本服务传参错误，也可能是远程服务器错误
	//客户端错误
	ClientErr                 = Add(4000, "client error")
	ClientErrQuery            = Add(4002, "verify query params failed")
	ClientErrBody             = Add(4004, "verify body params failed")
	ClientErrHeader           = Add(4005, "verify header params failed")
	ClientErrUri              = Add(4006, "verify uri params failed")
	ClientErr404              = Add(4040, "not found")
	ClientErrForbidConcurrent = Add(4201, "forbid concurrent by same user")
)
