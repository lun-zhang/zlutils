package code

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"reflect"
)

func WrapApi(api interface{}) gin.HandlerFunc {
	fv := reflect.ValueOf(api)
	ft := reflect.TypeOf(api)
	entry := logrus.WithFields(logrus.Fields{
		"ft": fmt.Sprintf("%s", ft),
		"fv": fmt.Sprintf("%s", fv),
	})

	if ft.Kind() != reflect.Func { //api必须是函数
		entry.Fatalf("api kind:%s isn't func", ft.Kind())
	}

	if ft.NumOut() != 2 { //出参个数必须为2
		entry.Fatalf("ft.NumOut():%d isn't 2", ft.NumOut())
	}

	errType := ft.Out(1)
	if _, ok := reflect.New(errType).Interface().(*error); !ok { //第二个出参必须是error类型
		entry.Fatalf("out[2] type:%s isn't error", errType.Name())
	}

	var reqValue reflect.Value
	var reqFieldMap map[string]reflect.Type
	if ft.NumIn() > 0 {
		if ft.NumIn() != 1 { //入参如果有就必须为1个
			entry.Fatalf("ft.NumIn():%d isn't 0 or 1", ft.NumIn())
		}
		reqType := ft.In(0)
		if reqType.Kind() != reflect.Struct {
			entry.Fatalf("req kind:%s isn't struct", reqType.Kind())
		}
		reqValue = reflect.New(reqType).Elem()
		reqFieldMap = map[string]reflect.Type{}
		for i := 0; i < reqType.NumField(); i++ {
			ti := reqType.Field(i)
			reqFieldMap[ti.Name] = ti.Type
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

		//响应
		var in []reflect.Value
		if reqValue.IsValid() { //如果有入参
			in = append(in, reqValue)
		}
		out := fv.Call(in)
		var err error
		if !out[1].IsNil() {
			err = out[1].Interface().(error)
		}
		Send(c, out[0].Interface(), err)
	}
}

var (
	//允许用户自定义
	ReqFieldNameBody   = "Body"
	ReqFieldNameQuery  = "Query"
	ReqFieldNameUri    = "Uri"
	ReqFieldNameHeader = "Header"
)
