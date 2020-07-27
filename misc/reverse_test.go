package misc

import (
	"fmt"
	"testing"
)

func TestReverseSlice(t *testing.T) {
	i32 := []int32{1, 3, 2, 4}

	ReverseSlice(i32)
	fmt.Println(i32)

	f64 := []float64{1., 3., 2., 4.}
	ReverseSlice(f64)
	fmt.Println(f64)
}

func TestReverseSliceWithSwap(t *testing.T) {
	i32 := []int32{1, 3, 2, 4}

	ReverseSliceWithSwap(i32, func(i, j int) {
		i32[i], i32[j] = i32[j], i32[i]
	})
	fmt.Println(i32)

	f64 := []float64{1., 3., 2., 4.}
	ReverseSliceWithSwap(f64, func(i, j int) {
		f64[i], f64[j] = f64[j], f64[i]
	})
	fmt.Println(f64)
}
