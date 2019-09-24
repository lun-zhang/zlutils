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

	var (
		reqType       reflect.Type
		reqBodyType   reflect.Type
		reqQueryType  reflect.Type
		reqUriType    reflect.Type
		reqHeaderType reflect.Type
		reqMetaType   reflect.Type
	)
	if numIn == 2 {
		reqType = ft.In(1)
		if reqType.Kind() != reflect.Struct {
			entry.Fatalf("req kind:%s isn't struct", reqType.Kind())
		}
		for i := 0; i < reqType.NumField(); i++ {
			ti := reqType.Field(i)
			switch ti.Name {
			case ReqFieldNameBody:
				reqBodyType = ti.Type
			case ReqFieldNameQuery:
				reqQueryType = ti.Type
			case ReqFieldNameUri:
				reqUriType = ti.Type
			case ReqFieldNameHeader:
				reqHeaderType = ti.Type
			case ReqFieldNameMeta:
				reqMetaType = ti.Type
				if reqMetaType.Kind() != reflect.Map {
					entry.Fatalf("reqMetaType kind:%s isn't map", reqMetaType.Kind())
				}
				if reqMetaType.Key().Kind() != reflect.String {
					entry.Fatalf("reqMetaType map key kind:%s isn't string", reqMetaType.Key().Kind())
				}
				if reqMetaType.Elem().Kind() != reflect.Interface {
					entry.Fatalf("reqMetaType map value kind:%s isn't interface{}", reqMetaType.Elem().Kind())
				}
			default: //在启动时就把非法的字段暴露出来，避免请求到了才知道字段定义错了
				entry.Fatalf("invalid req field name:%s", ti.Name)
			}
		}
	}

	return func(c *gin.Context) {
		//处理请求参数
		in := []reflect.Value{reflect.ValueOf(c.Request.Context())}
		if reqType != nil {
			reqValue := reflect.New(reqType).Elem()
			if reqBodyType != nil {
				bodyPtr := reflect.New(reqBodyType).Interface()
				if err := c.ShouldBindJSON(bodyPtr); err != nil {
					code.Send(c, nil, code.ClientErrBody.WithError(err))
					c.Abort()
					return
				}
				reqValue.FieldByName(ReqFieldNameBody).Set(reflect.ValueOf(bodyPtr).Elem())
			}
			if reqQueryType != nil {
				queryPtr := reflect.New(reqQueryType).Interface()
				if err := c.ShouldBindQuery(queryPtr); err != nil {
					code.Send(c, nil, code.ClientErrQuery.WithError(err))
					c.Abort()
					return
				}
				reqValue.FieldByName(ReqFieldNameQuery).Set(reflect.ValueOf(queryPtr).Elem())
			}
			if reqUriType != nil {
				uriPtr := reflect.New(reqUriType).Interface()
				if err := c.ShouldBindUri(uriPtr); err != nil {
					code.Send(c, nil, code.ClientErrUri.WithError(err))
					c.Abort()
					return
				}
				reqValue.FieldByName(ReqFieldNameUri).Set(reflect.ValueOf(uriPtr).Elem())
			}
			if reqHeaderType != nil {
				headerPtr := reflect.New(reqHeaderType).Interface()
				if err := ShouldBindHeader(c.Request.Header, headerPtr); err != nil {
					code.Send(c, nil, code.ClientErrHeader.WithError(err))
					c.Abort()
					return
				}
				reqValue.FieldByName(ReqFieldNameHeader).Set(reflect.ValueOf(headerPtr).Elem())
			}
			if reqMetaType != nil {
				reqValue.FieldByName(ReqFieldNameMeta).Set(reflect.ValueOf(c.Keys))
			}
			in = append(in, reqValue)
		}

		//响应
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
		code.Send(c, resp, err)
	}
}

const (
	ReqFieldNameBody   = "Body"
	ReqFieldNameQuery  = "Query"
	ReqFieldNameUri    = "Uri"
	ReqFieldNameHeader = "Header"
	ReqFieldNameMeta   = "Meta"
)
