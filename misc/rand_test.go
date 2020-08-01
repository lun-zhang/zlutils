package misc

import (
	"fmt"
	"testing"
)

func TestRandIntMinMax(t *testing.T) {
	for i := 0; i < 10; i++ {
		fmt.Print(RandIntMinMax(3, 5), " ")
	}
	fmt.Println()

	fmt.Println(RandIntMinMax(3, 3))

	for i := 0; i < 10; i++ {
		fmt.Print(RandIntMinMax(5, 3), " ")
	}
	fmt.Println()

	for i := 0; i < 10; i++ {
		fmt.Print(RandIntMinMax(-5, -3), " ")
	}
	fmt.Println()
}
