package misc

import (
	"reflect"
)

func ReverseSlice(slice interface{}, swap func(i, j int)) {
	v := reflect.ValueOf(slice)
	n := v.Len()
	for i := 0; i < n/2; i++ {
		j := n - i - 1
		swap(i, j)
	}
}

func ReverseInts(slice []int) {
	ReverseSlice(slice, func(i, j int) { slice[i], slice[j] = slice[j], slice[i] })
}

func ReverseInt8s(slice []int8) {
	ReverseSlice(slice, func(i, j int) { slice[i], slice[j] = slice[j], slice[i] })
}

func ReverseInt32s(slice []int32) {
	ReverseSlice(slice, func(i, j int) { slice[i], slice[j] = slice[j], slice[i] })
}

func ReverseInt64s(slice []int64) {
	ReverseSlice(slice, func(i, j int) { slice[i], slice[j] = slice[j], slice[i] })
}

func ReverseUints(slice []uint) {
	ReverseSlice(slice, func(i, j int) { slice[i], slice[j] = slice[j], slice[i] })
}

func ReverseUint8s(slice []uint8) {
	ReverseSlice(slice, func(i, j int) { slice[i], slice[j] = slice[j], slice[i] })
}

func ReverseUint32s(slice []uint32) {
	ReverseSlice(slice, func(i, j int) { slice[i], slice[j] = slice[j], slice[i] })
}

func ReverseUint64s(slice []uint64) {
	ReverseSlice(slice, func(i, j int) { slice[i], slice[j] = slice[j], slice[i] })
}

func ReverseFloat32s(slice []float32) {
	ReverseSlice(slice, func(i, j int) { slice[i], slice[j] = slice[j], slice[i] })
}

func ReverseFloat64s(slice []float64) {
	ReverseSlice(slice, func(i, j int) { slice[i], slice[j] = slice[j], slice[i] })
}
