package mysql

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"reflect"
	"zlutils/misc"
)

func SetZero(i interface{}) (err error) {
	iv := reflect.ValueOf(i)
	if iv.Kind() != reflect.Ptr { //NOTE: 自己写的函数就要避免panic
		return fmt.Errorf("set zero faild: %v is't ptr", iv.Kind())
	}
	iv.Elem().Set(reflect.Zero(iv.Elem().Type()))
	return
}

func Value(i interface{}) (driver.Value, error) {
	if misc.IsNil(i) { //NOTE: 如果是nil则插入到数据库NULL
		return nil, nil
	}
	return json.Marshal(i) //struct类型不会为nil，即使是零值也会插入到数据库
}

func Scan(dest, src interface{}) (err error) {
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

type (
	SS       []string
	IntSlice []int
	MSS      map[string]string
	MSI      map[string]interface{}
)

func (j SS) Value() (driver.Value, error) {
	return Value(j)
}

func (j *SS) Scan(src interface{}) error {
	return Scan(j, src)
}
func (j IntSlice) Value() (driver.Value, error) {
	return Value(j)
}

func (j *IntSlice) Scan(src interface{}) error {
	return Scan(j, src)
}

func (j MSS) Value() (driver.Value, error) {
	return Value(j)
}

func (j *MSS) Scan(src interface{}) error {
	return Scan(j, src)
}

func (j MSI) Value() (driver.Value, error) {
	return Value(j)
}

func (j *MSI) Scan(src interface{}) error {
	return Scan(j, src)
}
