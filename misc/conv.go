package misc

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"reflect"
)

func GetFieldNameFromSlice(ctx context.Context, slice interface{}, fieldName string, recvSlicePtr interface{}) (err error) {
	entry := logrus.WithContext(ctx).
		WithFields(logrus.Fields{
			"slice":        slice,
			"fieldName":    fieldName,
			"recvSlicePtr": recvSlicePtr,
		})
	v := reflect.ValueOf(slice)
	if v.Kind() != reflect.Slice {
		err = fmt.Errorf("slice kind:%s isnt slice", v.Kind())
		entry.WithError(err).Error()
		return
	}
	recvSlicePtrV := reflect.ValueOf(recvSlicePtr)
	if recvSlicePtrV.Kind() != reflect.Ptr {
		err = fmt.Errorf("recvSlicePtrV kind:%s isnt ptr", recvSlicePtrV.Kind())
		entry.WithError(err).Error()
		return
	}
	recvSlice := recvSlicePtrV.Elem()
	if recvSlice.Kind() != reflect.Slice {
		err = fmt.Errorf("recvSlice kind:%s isnt slice", recvSlice.Kind())
		entry.WithError(err).Error()
		return
	}
	recvSliceNew := reflect.MakeSlice(recvSlice.Type(), v.Len(), v.Len())
	elemType := recvSlice.Type().Elem()

	for i := 0; i < v.Len(); i++ {
		elem := v.Index(i)
		if elem.Kind() != reflect.Struct {
			err = fmt.Errorf("elem kind:%s isnt struct", elem.Kind())
			entry.WithError(err).Error()
			return
		}
		field := elem.FieldByName(fieldName)
		if !field.IsValid() {
			err = fmt.Errorf("fieldName:%s invalid", fieldName)
			entry.WithError(err).Error()
			return
		}
		if elemType != field.Type() {
			err = fmt.Errorf("elem type:%s isnt field type:%s", elemType, field.Type())
			entry.WithError(err).Error()
			return
		}
		recvSliceNew.Index(i).Set(field)
	}
	recvSlice.Set(recvSliceNew)
	return
}

//将数组转成map，指定一个field当做key
//如果有重复的key，则err
func ConvStructSliceToMap(ctx context.Context, slice interface{}, keyFiledName string, recvMapPtr interface{}) (err error) {
	entry := logrus.WithContext(ctx).
		WithFields(logrus.Fields{
			"slice":        slice,
			"keyFiledName": keyFiledName,
			"recvMapPtr":   recvMapPtr,
		})
	v := reflect.ValueOf(slice)
	if v.Kind() != reflect.Slice {
		err = fmt.Errorf("slice kind:%s isnt slice", v.Kind())
		entry.WithError(err).Error()
		return
	}
	recvMapPtrV := reflect.ValueOf(recvMapPtr)
	if recvMapPtrV.Kind() != reflect.Ptr {
		err = fmt.Errorf("recvMapPtrV kind:%s isnt ptr", recvMapPtrV.Kind())
		entry.WithError(err).Error()
		return
	}
	recvMap := recvMapPtrV.Elem()
	if recvMap.Kind() != reflect.Map {
		err = fmt.Errorf("recvSrecvMaplice kind:%s isnt slice", recvMap.Kind())
		entry.WithError(err).Error()
		return
	}
	recvMapNew := reflect.MakeMap(recvMap.Type())
	keyType := recvMap.Type().Key()
	valueType := recvMap.Type().Elem()

	for i := 0; i < v.Len(); i++ {
		elem := v.Index(i)
		if elem.Kind() != reflect.Struct {
			err = fmt.Errorf("elem kind:%s isnt struct", elem.Kind())
			entry.WithError(err).Error()
			return
		}
		if elem.Type() != valueType {
			err = fmt.Errorf("map value type:%s isnt elem type:%s", valueType, elem.Type())
			entry.WithError(err).Error()
			return
		}
		keyField := elem.FieldByName(keyFiledName)
		if !keyField.IsValid() {
			err = fmt.Errorf("keyFiledName:%s invalid", keyFiledName)
			entry.WithError(err).Error()
			return
		}
		if keyType != keyField.Type() {
			err = fmt.Errorf("map key type:%s isnt keyField type:%s", keyType, keyField.Type())
			entry.WithError(err).Error()
			return
		}
		if recvMapNew.MapIndex(keyField).IsValid() {
			err = fmt.Errorf("duplicate key:%v", keyField)
			entry.WithError(err).Error()
			return
		}
		recvMapNew.SetMapIndex(keyField, elem)
	}
	recvMap.Set(recvMapNew)
	return
}
