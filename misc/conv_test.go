package misc

import (
	"context"
	"fmt"
	"github.com/aws/aws-xray-sdk-go/xray"
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

var ctx, _ = xray.BeginSegment(context.Background(), "test")

func TestCopyFieldNameFromSlice(t *testing.T) {
	var ids []int
	GetFieldNameFromSlice(ctx, as, "Id", &ids)
	fmt.Println(ids)

	var names []string
	GetFieldNameFromSlice(ctx, as, "Name", &names)
	fmt.Println(names)
}

func TestConvStructSliceToMap(t *testing.T) {
	var idM map[int]A
	if err := ConvStructSliceToMap(ctx, as, "Id", &idM); err != nil {
		t.Fatal(err)
	}
	fmt.Println(idM)

	var nameM map[string]A
	if err := ConvStructSliceToMap(ctx, as, "Name", &nameM); err != nil {
		t.Fatal(err)
	}
	fmt.Println(nameM)
}
