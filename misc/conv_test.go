package misc

import (
	"fmt"
	"testing"
)

type A struct {
	Id   int
	Name string
}

var as = []A{
	{
		Id:   1,
		Name: "a",
	},
	{
		Id:   2,
		Name: "b",
	},
	{
		Id:   3,
		Name: "b",
	},
}

func TestCopyFieldNameFromSlice(t *testing.T) {
	var ids []int
	GetFieldNameFromSlice(as, "Id", &ids)
	fmt.Println(ids)

	var names []string
	GetFieldNameFromSlice(as, "Name", &names)
	fmt.Println(names)
}

func TestConvStructSliceToMap(t *testing.T) {
	var idM map[int]A
	if err := ConvStructSliceToMap(as, "Id", &idM); err != nil {
		t.Fatal(err)
	}
	fmt.Println(idM)

	var nameM map[string]A
	if err := ConvStructSliceToMap(as, "Name", &nameM); err != nil {
		t.Fatal(err)
	}
	fmt.Println(nameM)
}
