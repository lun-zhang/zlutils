package code

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/sirupsen/logrus"
	"net/http"
	"reflect"
	"strconv"
	"time"
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
		entry.Fatalf("out(1) type:%s isn't error", errType.Name())
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
			if err := bindHeader(c.Request.Header, headerPtr); err != nil {
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
		var err error
		if !out[1].IsNil() {
			err = out[1].Interface().(error)
		}
		Send(c, out[0].Interface(), err)
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

func bindHeader(header http.Header, obj interface{}) (err error) {
	if header == nil || obj == nil {
		return fmt.Errorf("invalid request")
	}
	v := reflect.ValueOf(obj)
	if v.Kind() != reflect.Ptr {
		return fmt.Errorf("obj kind:%s isn't ptr", v.Kind())
	}
	ve := v.Elem()
	if ve.Kind() != reflect.Struct {
		return fmt.Errorf("obj elem:%s isn't struct", ve.Kind())
	}
	te := reflect.TypeOf(ve.Interface())
	defer func() {
		if err != nil {
			ve.Set(reflect.Zero(te)) //注意：发生错误时，要重置为零值，否则可能出现隐晦bug
		}
	}()
	for i := 0; i < te.NumField(); i++ {
		ti := te.Field(i)
		vi := ve.Field(i)
		name := ti.Tag.Get("header")
		if name == "-" {
			continue
		}
		if name == "" { //没有tag，默认为field name
			name = ti.Name
		}
		s := header.Get(name)
		if err = setWithProperType(s, vi, ti); err != nil {
			return
		}
	}
	return binding.Validator.ValidateStruct(ve.Interface())
}

//从 github.com/gin-gonic/gin@v1.4.0/binding/form_mapping.go:170 复制过来的
func setWithProperType(val string, value reflect.Value, field reflect.StructField) error {
	switch value.Kind() {
	case reflect.Int:
		return setIntField(val, 0, value)
	case reflect.Int8:
		return setIntField(val, 8, value)
	case reflect.Int16:
		return setIntField(val, 16, value)
	case reflect.Int32:
		return setIntField(val, 32, value)
	case reflect.Int64:
		switch value.Interface().(type) {
		case time.Duration:
			return setTimeDuration(val, value, field)
		}
		return setIntField(val, 64, value)
	case reflect.Uint:
		return setUintField(val, 0, value)
	case reflect.Uint8:
		return setUintField(val, 8, value)
	case reflect.Uint16:
		return setUintField(val, 16, value)
	case reflect.Uint32:
		return setUintField(val, 32, value)
	case reflect.Uint64:
		return setUintField(val, 64, value)
	case reflect.Bool:
		return setBoolField(val, value)
	case reflect.Float32:
		return setFloatField(val, 32, value)
	case reflect.Float64:
		return setFloatField(val, 64, value)
	case reflect.String:
		value.SetString(val)
	case reflect.Struct:
		switch value.Interface().(type) {
		case time.Time:
			return setTimeField(val, field, value)
		}
		return json.Unmarshal([]byte(val), value.Addr().Interface())
	case reflect.Map:
		return json.Unmarshal([]byte(val), value.Addr().Interface())
	default:
		return fmt.Errorf("unsport kind:%s", value.Kind())
	}
	return nil
}

func setIntField(val string, bitSize int, field reflect.Value) error {
	if val == "" {
		val = "0"
	}
	intVal, err := strconv.ParseInt(val, 10, bitSize)
	if err == nil {
		field.SetInt(intVal)
	}
	return err
}

func setUintField(val string, bitSize int, field reflect.Value) error {
	if val == "" {
		val = "0"
	}
	uintVal, err := strconv.ParseUint(val, 10, bitSize)
	if err == nil {
		field.SetUint(uintVal)
	}
	return err
}

func setBoolField(val string, field reflect.Value) error {
	if val == "" {
		val = "false"
	}
	boolVal, err := strconv.ParseBool(val)
	if err == nil {
		field.SetBool(boolVal)
	}
	return err
}

func setFloatField(val string, bitSize int, field reflect.Value) error {
	if val == "" {
		val = "0.0"
	}
	floatVal, err := strconv.ParseFloat(val, bitSize)
	if err == nil {
		field.SetFloat(floatVal)
	}
	return err
}

func setTimeField(val string, structField reflect.StructField, value reflect.Value) error {
	timeFormat := structField.Tag.Get("time_format")
	if timeFormat == "" {
		timeFormat = time.RFC3339
	}

	if val == "" {
		value.Set(reflect.ValueOf(time.Time{}))
		return nil
	}

	l := time.Local
	if isUTC, _ := strconv.ParseBool(structField.Tag.Get("time_utc")); isUTC {
		l = time.UTC
	}

	if locTag := structField.Tag.Get("time_location"); locTag != "" {
		loc, err := time.LoadLocation(locTag)
		if err != nil {
			return err
		}
		l = loc
	}

	t, err := time.ParseInLocation(timeFormat, val, l)
	if err != nil {
		return err
	}

	value.Set(reflect.ValueOf(t))
	return nil
}

func setArray(vals []string, value reflect.Value, field reflect.StructField) error {
	for i, s := range vals {
		err := setWithProperType(s, value.Index(i), field)
		if err != nil {
			return err
		}
	}
	return nil
}

func setSlice(vals []string, value reflect.Value, field reflect.StructField) error {
	slice := reflect.MakeSlice(value.Type(), len(vals), len(vals))
	err := setArray(vals, slice, field)
	if err != nil {
		return err
	}
	value.Set(slice)
	return nil
}

func setTimeDuration(val string, value reflect.Value, field reflect.StructField) error {
	d, err := time.ParseDuration(val)
	if err != nil {
		return err
	}
	value.Set(reflect.ValueOf(d))
	return nil
}
