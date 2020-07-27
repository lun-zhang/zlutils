package misc

import (
	"fmt"
	"testing"
)

func TestReverseSlice(t *testing.T) {
	a := []int32{1, 3, 2, 4}

	ReverseSlice(a, func(i, j int) {
		a[i], a[j] = a[j], a[i]
	})
	fmt.Println(a)
}
