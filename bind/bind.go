package bind

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/sirupsen/logrus"
	"reflect"
	"strings"
	"zlutils/caller"
	"zlutils/code"
)

func isErrType(t reflect.Type) (ok bool) {
	_, ok = reflect.New(t).Interface().(*error)
	return
}

// 检查请求结构(出现匿名成员时会递归进入)
// 不允许相同的成员名（例如Body）出现多次，
// 也不允许出现Body Query Header Uri Meta之外的名称，
// 不需要这里限制Query Header Uri必须是struct，因为下面gin的bind会检查出来，
// 但是需要这里检测Meta的类型必须是map[string]interface
func checkReqType(t reflect.Type, has map[string]struct{}) {
	if t.Kind() != reflect.Struct {
		logrus.Fatalf("req kind:%s isn't struct", t.Kind())
	}
	for i := 0; i < t.NumField(); i++ {
		ti := t.Field(i)
		if ti.Anonymous {
			checkReqType(ti.Type, has)
		} else {
			if _, ok := has[ti.Name]; ok {
				logrus.Fatalf("req field name:%s appear twice", ti.Name)
			}
			has[ti.Name] = struct{}{}
			switch ti.Name {
			case ReqFieldNameBody:
			case ReqFieldNameQuery:
			case ReqFieldNameUri:
			case ReqFieldNameHeader:
			case reqFieldNameMeta:
			case reqFieldNameC:
			default: //在启动时就把非法的字段暴露出来，避免请求到了才知道字段定义错了
				logrus.Fatalf("invalid req field name:%s", ti.Name)
			}
		}
	}
}

type wrapper struct {
	reqErrSender reqErrSenderFunc
	resultSender resultSenderFunc
}

func (m wrapper) Wrap(api interface{}) gin.HandlerFunc {
	return wrap(api, m.reqErrSender, m.resultSender)
}

func Wrap(api interface{}) gin.HandlerFunc {
	return wrap(api, nil, nil)
}
func wrap(api interface{}, reqErrSender reqErrSenderFunc, resultSender resultSenderFunc) gin.HandlerFunc {
	fv := reflect.ValueOf(api)
	ft := reflect.TypeOf(api)
	entry := logrus.WithFields(logrus.Fields{
		"ft":     ft.String(),
		"caller": caller.Caller(2),
	})

	if ft.Kind() != reflect.Func { //api必须是函数
		entry.Fatalf("api kind:%s isn't func", ft.Kind())
	}

	numOut := ft.NumOut()
	if resultSender != nil {
		if numOut != 2 { //当自定义sender时，必须只能有2个出参
			entry.Fatalf("numOut:%d ne 2 when with sender", numOut)
		}
		if codeType := ft.Out(0); codeType.Kind() != reflect.Int {
			entry.Fatalf("out(0) kind:%s isn't int", codeType.Kind())
		}
	} else {
		if numOut > 2 {
			entry.Fatalf("numOut:%d bigger than 2", numOut)
		}

		if numOut == 2 {
			if errType := ft.Out(1); !isErrType(errType) { //第二个出参必须是error类型
				entry.Fatalf("out(1) type:%s isn't error", errType.Name())
			}
		}
	}

	numIn := ft.NumIn()
	if numIn != 1 && numIn != 2 { //入参只能是(ctx)或(ctx,req)
		entry.Fatalf("numIn:%d isn't in 1 or 2", numIn)
	}

	ctxType := ft.In(0)
	if _, ok := reflect.New(ctxType).Interface().(*context.Context); !ok { //第1个入参必须是context类型
		entry.Fatalf("in(0) type:%s isn't context.Context", ctxType.Name())
	}

	var reqType reflect.Type
	if numIn == 2 {
		reqType = ft.In(1)
		checkReqType(reqType, map[string]struct{}{})
	}

	return func(c *gin.Context) {
		ctx := c.Request.Context()
		entry := logrus.WithContext(ctx)
		//处理请求参数
		in := []reflect.Value{reflect.ValueOf(ctx)}
		if reqType != nil {
			reqValue, reqFieldName, err := shouldBindReq(c, reqType)
			if err != nil {
				if reqErrSender != nil {
					reqErrSender(c, reqFieldName, err)
				} else {
					switch reqFieldName {
					case ReqFieldNameBody:
						err = code.ClientErrBody.WithError(err)
					case ReqFieldNameQuery:
						err = code.ClientErrQuery.WithError(err)
					case ReqFieldNameUri:
						err = code.ClientErrUri.WithError(err)
					case ReqFieldNameHeader:
						err = code.ClientErrHeader.WithError(err)
					}
					entry.WithError(err).Warn()
					code.Send(c, nil, err)
				}
				c.Abort()
				return
			}
			in = append(in, reqValue)
		}

		//响应
		out := fv.Call(in)

		if resultSender != nil {
			_code := out[0].Interface().(int)
			obj := out[1].Interface()
			resultSender(c, _code, obj) //有resultSender时前面限制了必定有一个出参
		} else {
			var resp interface{}
			var err error

			switch numOut {
			case 1: //NOTE: 返回只有一个参数的时候，如果是error类型则被认为是err，因此如果想要让返回err类型的resp时候，必须用2个返回参数(resp,err)
				if isErrType(out[0].Type()) { //(err)
					if !out[0].IsNil() { //nil.(error)会panic
						err = out[0].Interface().(error)
					}
				} else { //(resp)
					resp = out[0].Interface()
				}
			case 2: //(resp,err)
				resp = out[0].Interface()
				if !out[1].IsNil() {
					err = out[1].Interface().(error)
				}
			}
			code.Send(c, resp, err)
		}
	}
}

var bindFuncs = []struct {
	name     string
	bindFunc func(c *gin.Context, obj interface{}, tagMap map[string]struct{}) (err error)
}{
	{
		name: ReqFieldNameBody,
		bindFunc: func(c *gin.Context, obj interface{}, tagMap map[string]struct{}) (err error) {
			if _, ok := tagMap[tagReuseBody]; ok {
				err = c.ShouldBindBodyWith(obj, binding.JSON)
			} else {
				err = c.ShouldBindJSON(obj)
			}
			return
		},
	},

	{
		name: ReqFieldNameQuery,
		bindFunc: func(c *gin.Context, obj interface{}, tagMap map[string]struct{}) (err error) {
			return c.ShouldBindQuery(obj)
		},
	},
	{
		name: ReqFieldNameUri,
		bindFunc: func(c *gin.Context, obj interface{}, tagMap map[string]struct{}) (err error) {
			return c.ShouldBindUri(obj)
		},
	},
	{
		name: ReqFieldNameHeader,
		bindFunc: func(c *gin.Context, obj interface{}, tagMap map[string]struct{}) (err error) {
			return c.ShouldBindHeader(obj)
		},
	},
}

func shouldBindReq(c *gin.Context, reqType reflect.Type) (reqValue reflect.Value, reqFieldName string, err error) {
	reqValue = reflect.New(reqType).Elem()

	for _, bf := range bindFuncs {
		reqFieldName = bf.name
		if fieldType, ok := reqType.FieldByName(bf.name); ok {
			fieldValuePtr := reflect.New(fieldType.Type).Interface()

			tagMap := map[string]struct{}{}
			for _, tag := range strings.Split(fieldType.Tag.Get(tagKey), ",") {
				tagMap[tag] = struct{}{}
			}

			if err = bf.bindFunc(c, fieldValuePtr, tagMap); err != nil {
				if _, ok := tagMap[tagIgnoreError]; ok {
					err = nil //如果发生了错误，但是有ignore_error标签，那么就继续，也没有warn日志
				} else {
					return
				}
			} else {
				reqValue.FieldByIndex(fieldType.Index).Set(reflect.ValueOf(fieldValuePtr).Elem())
			}
		}
	}

	if fieldType, ok := reqType.FieldByName(reqFieldNameMeta); ok {
		reqValue.FieldByIndex(fieldType.Index).Set(reflect.ValueOf(c.Keys))
	}
	if fieldType, ok := reqType.FieldByName(reqFieldNameC); ok {
		reqValue.FieldByIndex(fieldType.Index).Set(reflect.ValueOf(c))
	}
	return
}

const (
	ReqFieldNameBody   = "Body"
	ReqFieldNameQuery  = "Query"
	ReqFieldNameUri    = "Uri"
	ReqFieldNameHeader = "Header"
	reqFieldNameMeta   = "Meta"
	reqFieldNameC      = "C"
)

const (
	tagKey         = "bind"
	tagReuseBody   = "reuse_body"
	tagIgnoreError = "ignore_error"
)

type reqErrSenderFunc func(c *gin.Context, reqFieldName string, bindErr error)

//后两个入参参数就是c.JSON的两个入参
type resultSenderFunc func(c *gin.Context, code int, obj interface{})

func WithSender(reqErrSender reqErrSenderFunc, resultSender resultSenderFunc) wrapper {
	return wrapper{
		resultSender: resultSender,
		reqErrSender: reqErrSender,
	}
}

//func WithReqBindErrSender(reqErrSender reqErrSenderFunc) wrapper {
//	return wrapper{}.WithReqBindErrSender(reqErrSender)
//}
//
//func WithResultSender(resultSender resultSenderFunc) wrapper {
//	return wrapper{}.WithResultSender(resultSender)
//}
//
//func (m wrapper) WithReqBindErrSender(reqErrSender reqErrSenderFunc) wrapper {
//	m.reqErrSender = reqErrSender
//	return m
//}
//
//func (m wrapper) WithResultSender(resultSender resultSenderFunc) wrapper {
//	m.resultSender = resultSender
//	return m
//}
