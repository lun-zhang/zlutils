package misc

import "reflect"

func AbsInt(i int) int {
	if i >= 0 {
		return i
	}
	return -i
}

func AbsInt32(i int32) int32 {
	if i >= 0 {
		return i
	}
	return -i
}

func min(a, b interface{}) interface{} {
	va := reflect.ValueOf(a)
	vb := reflect.ValueOf(b)
	switch va.Kind() {
	case reflect.Uint:
	case reflect.Uint8:
	case reflect.Uint16:
	case reflect.Uint32:
	case reflect.Uint64:
		for i := 0; i < vb.Len(); i++ {
			bi := vb.Index(i)
			if va.Uint() > bi.Uint() {
				va = bi
			}
		}
	default:
		for i := 0; i < vb.Len(); i++ {
			bi := vb.Index(i)
			if va.Int() > bi.Int() {
				va = bi
			}
		}
	}
	return va.Interface()
}

func max(a, b interface{}) interface{} {
	va := reflect.ValueOf(a)
	vb := reflect.ValueOf(b)
	switch va.Kind() {
	case reflect.Uint:
	case reflect.Uint8:
	case reflect.Uint16:
	case reflect.Uint32:
	case reflect.Uint64:
		for i := 0; i < vb.Len(); i++ {
			bi := vb.Index(i)
			if va.Uint() < bi.Uint() {
				va = bi
			}
		}
	default:
		for i := 0; i < vb.Len(); i++ {
			bi := vb.Index(i)
			if va.Int() < bi.Int() {
				va = bi
			}
		}
	}
	return va.Interface()
}

func MinInt(a int, b ...int) int { return min(a, b).(int) }
func MaxInt(a int, b ...int) int { return max(a, b).(int) }

func MinInt8(a int8, b ...int8) int8 { return min(a, b).(int8) }
func MaxInt8(a int8, b ...int8) int8 { return max(a, b).(int8) }

func MinInt16(a int16, b ...int16) int16 { return min(a, b).(int16) }
func MaxInt16(a int16, b ...int16) int16 { return max(a, b).(int16) }

func MinInt64(a int64, b ...int64) int64 { return min(a, b).(int64) }
func MaxInt64(a int64, b ...int64) int64 { return max(a, b).(int64) }

func MinUint(a uint, b ...uint) uint { return min(a, b).(uint) }
func MaxUint(a uint, b ...uint) uint { return max(a, b).(uint) }

func MinUint8(a uint8, b ...uint8) uint8 { return min(a, b).(uint8) }
func MaxUint8(a uint8, b ...uint8) uint8 { return max(a, b).(uint8) }

func MinUint16(a uint16, b ...uint16) uint16 { return min(a, b).(uint16) }
func MaxUint16(a uint16, b ...uint16) uint16 { return max(a, b).(uint16) }

func MinUint64(a uint64, b ...uint64) uint64 { return min(a, b).(uint64) }
func MaxUint64(a uint64, b ...uint64) uint64 { return max(a, b).(uint64) }
