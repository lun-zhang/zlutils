package misc

import (
	"reflect"
)

func ReverseSlice(slice interface{}) {
	v := reflect.ValueOf(slice)
	n := v.Len()
	for i := 0; i < n/2; i++ {
		j := n - i - 1
		t := v.Index(i).Interface()
		v.Index(i).Set(v.Index(j))
		v.Index(j).Set(reflect.ValueOf(t))
	}
}

func ReverseSliceWithSwap(slice interface{}, swap func(i, j int)) {
	v := reflect.ValueOf(slice)
	n := v.Len()
	for i := 0; i < n/2; i++ {
		j := n - i - 1
		swap(i, j)
	}
}
