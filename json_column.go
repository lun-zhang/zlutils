package zlutils

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"reflect"
)

func SetZero(i interface{}) (err error) {
	iv := reflect.ValueOf(i)
	if iv.Kind() != reflect.Ptr { //NOTE: 自己写的函数就要避免panic
		return fmt.Errorf("set zero faild: %v is't ptr", iv.Kind())
	}
	iv.Elem().Set(reflect.Zero(iv.Elem().Type()))
	return
}

func IsPtrNil(i interface{}) (ok bool) {
	defer func() {
		if err := recover(); err != nil {
			ok = false
		}
	}()
	return reflect.ValueOf(i).IsNil() //TODO 非指针类型会panic，所以recover并返回false，是否有更好的做法？
}

func JCValue(i interface{}) (driver.Value, error) {
	if IsPtrNil(i) { //NOTE: 如果是nil则插入到数据库NULL
		return nil, nil
	}
	return json.Marshal(i) //struct类型不会为nil，即使是零值也会插入到数据库
}

func JCScan(dest, src interface{}) (err error) {
	if err = SetZero(dest); err != nil {
		return
	}
	if src == nil {
		return nil
	}
	bs, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("src must be []byte")
	}
	return json.Unmarshal(bs, dest)
}

//json列，与数据库的[]byte转换，实现该接口的类型应统一以JC
type JsonColumn interface {
	driver.Valuer
	sql.Scanner
}

type (
	StringSliceJC     []string
	MapStringStringJC map[string]string
)

func (j StringSliceJC) Value() (driver.Value, error) {
	return JCValue(j)
}

func (j *StringSliceJC) Scan(src interface{}) error {
	return JCScan(j, src)
}

func (j MapStringStringJC) Value() (driver.Value, error) {
	return JCValue(j)
}

func (j *MapStringStringJC) Scan(src interface{}) error {
	return JCScan(j, src)
}
