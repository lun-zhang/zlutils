package bind

import (
	"context"
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
			case ReqFieldNameMeta:
			default: //在启动时就把非法的字段暴露出来，避免请求到了才知道字段定义错了
				logrus.Fatalf("invalid req field name:%s", ti.Name)
			}
		}
	}
}

func Wrap(api interface{}) gin.HandlerFunc {
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

	var reqType reflect.Type
	if numIn == 2 {
		reqType = ft.In(1)
		checkReqType(reqType, map[string]struct{}{})
	}

	return func(c *gin.Context) {
		//处理请求参数
		in := []reflect.Value{reflect.ValueOf(c.Request.Context())}
		if reqType != nil {
			reqValue := reflect.New(reqType).Elem()

			if body := reqValue.FieldByName(ReqFieldNameBody); body.IsValid() {
				bodyPtr := reflect.New(body.Type()).Interface()
				if err := c.ShouldBindJSON(bodyPtr); err != nil {
					code.Send(c, nil, code.ClientErrBody.WithError(err))
					c.Abort()
					return
				}
				body.Set(reflect.ValueOf(bodyPtr).Elem())
			}
			if query := reqValue.FieldByName(ReqFieldNameQuery); query.IsValid() {
				queryPtr := reflect.New(query.Type()).Interface()
				if err := c.ShouldBindQuery(queryPtr); err != nil {
					code.Send(c, nil, code.ClientErrQuery.WithError(err))
					c.Abort()
					return
				}
				query.Set(reflect.ValueOf(queryPtr).Elem())
			}
			if uri := reqValue.FieldByName(ReqFieldNameUri); uri.IsValid() {
				uriPtr := reflect.New(uri.Type()).Interface()
				if err := c.ShouldBindUri(uriPtr); err != nil {
					code.Send(c, nil, code.ClientErrUri.WithError(err))
					c.Abort()
					return
				}
				uri.Set(reflect.ValueOf(uriPtr).Elem())
			}
			if header := reqValue.FieldByName(ReqFieldNameHeader); header.IsValid() {
				headerPtr := reflect.New(header.Type()).Interface()
				if err := ShouldBindHeader(c.Request.Header, headerPtr); err != nil {
					code.Send(c, nil, code.ClientErrHeader.WithError(err))
					c.Abort()
					return
				}
				header.Set(reflect.ValueOf(headerPtr).Elem())
			}
			if meta := reqValue.FieldByName(ReqFieldNameMeta); meta.IsValid() {
				meta.Set(reflect.ValueOf(c.Keys))
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
