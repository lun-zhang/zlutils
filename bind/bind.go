package bind

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"reflect"
	"zlutils/caller"
	"zlutils/code"
)

func isErrType(t reflect.Type) (ok bool) {
	_, ok = reflect.New(t).Interface().(*error)
	return
}

func Wrap(api interface{}) gin.HandlerFunc {
	fv := reflect.ValueOf(api)
	ft := reflect.TypeOf(api)
	entry := logrus.WithFields(logrus.Fields{
		"ft":     fmt.Sprintf("%s", ft),
		"fv":     fmt.Sprintf("%s", fv),
		"caller": caller.Caller(2),
	})

	if ft.Kind() != reflect.Func { //api必须是函数
		entry.Fatalf("api kind:%s isn't func", ft.Kind())
	}

	numOut := ft.NumOut()
	if numOut > 2 {
		entry.Fatalf("numOut:%d bigger than 2", numOut)
	}

	if numOut == 2 {
		if errType := ft.Out(1); !isErrType(errType) { //第二个出参必须是error类型
			entry.Fatalf("out(1) type:%s isn't error", errType.Name())
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

	var reqValue reflect.Value
	var reqFieldMap map[string]reflect.Type
	if numIn == 2 {
		reqType := ft.In(1)
		if reqType.Kind() != reflect.Struct {
			entry.Fatalf("req kind:%s isn't struct", reqType.Kind())
		}
		reqValue = reflect.New(reqType).Elem()
		reqFieldMap = map[string]reflect.Type{}
		for i := 0; i < reqType.NumField(); i++ {
			ti := reqType.Field(i)
			reqFieldMap[ti.Name] = ti.Type
			if ti.Name == ReqFieldNameMeta {
				metaType := ti.Type
				if metaType.Kind() != reflect.Map {
					logrus.Fatalf("metaType kind:%s isn't map", metaType.Kind())
				}
				if metaType.Key().Kind() != reflect.String {
					logrus.Fatalf("metaType map key kind:%s isn't string", metaType.Key().Kind())
				}
				if metaType.Elem().Kind() != reflect.Interface {
					logrus.Fatalf("metaType map value kind:%s isn't interface{}", metaType.Elem().Kind())
				}
			}
		}
	}

	return func(c *gin.Context) {
		//处理请求参数
		if bodyType, ok := reqFieldMap[ReqFieldNameBody]; ok {
			bodyPtr := reflect.New(bodyType).Interface()
			if err := c.ShouldBindJSON(bodyPtr); err != nil {
				Send(c, nil, ClientErrBody.WithError(err))
				c.Abort()
				return
			}
			reqValue.FieldByName(ReqFieldNameBody).Set(reflect.ValueOf(bodyPtr).Elem())
		}
		if queryType, ok := reqFieldMap[ReqFieldNameQuery]; ok {
			queryPtr := reflect.New(queryType).Interface()
			if err := c.ShouldBindQuery(queryPtr); err != nil {
				Send(c, nil, ClientErrQuery.WithError(err))
				c.Abort()
				return
			}
			reqValue.FieldByName(ReqFieldNameQuery).Set(reflect.ValueOf(queryPtr).Elem())
		}
		if uriType, ok := reqFieldMap[ReqFieldNameUri]; ok {
			uriPtr := reflect.New(uriType).Interface()
			if err := c.ShouldBindUri(uriPtr); err != nil {
				Send(c, nil, ClientErrUri.WithError(err))
				c.Abort()
				return
			}
			reqValue.FieldByName(ReqFieldNameUri).Set(reflect.ValueOf(uriPtr).Elem())
		}
		if headerType, ok := reqFieldMap[ReqFieldNameHeader]; ok {
			headerPtr := reflect.New(headerType).Interface()
			if err := ShouldBindHeader(c.Request.Header, headerPtr); err != nil {
				Send(c, nil, ClientErrHeader.WithError(err))
				c.Abort()
				return
			}
			reqValue.FieldByName(ReqFieldNameHeader).Set(reflect.ValueOf(headerPtr).Elem())
		}
		if _, ok := reqFieldMap[ReqFieldNameMeta]; ok {
			reqValue.FieldByName(ReqFieldNameMeta).Set(reflect.ValueOf(c.Keys))
		}

		//响应
		var in []reflect.Value
		in = append(in, reflect.ValueOf(c.Request.Context()))
		if numIn == 2 {
			in = append(in, reqValue)
		}
		out := fv.Call(in)

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
		Send(c, resp, err)
	}
}

var (
	//允许用户自定义(其实也可以改成tag，但是不能用FieldByName了)
	ReqFieldNameBody   = "Body"
	ReqFieldNameQuery  = "Query"
	ReqFieldNameUri    = "Uri"
	ReqFieldNameHeader = "Header"
	ReqFieldNameMeta   = "Meta"
)

//允许用户自定义
var Send = code.Send
var (
	ClientErrHeader = code.ClientErrHeader
	ClientErrUri    = code.ClientErrUri
	ClientErrQuery  = code.ClientErrQuery
	ClientErrBody   = code.ClientErrBody
)
