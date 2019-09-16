package caller

import (
	"fmt"
	"testing"
)

func TestCaller(t *testing.T) {
	fmt.Println(Caller(0))
	fmt.Println(Caller(1))
	fmt.Println(Caller(2))
}

func TestStack(t *testing.T) {
	Init("zlutils")
	for _, v := range Stack(0) {
		fmt.Println(v)
	}
}
