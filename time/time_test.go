package time

import (
	"encoding/json"
	"fmt"
	"github.com/go-playground/validator"
	"reflect"
	"testing"
	"time"
)

func TestTime(t *testing.T) {
	a := Time{Time: time.Now()}
	bs, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(string(bs))

	var b Time
	if err := json.Unmarshal([]byte("1564576609"), &b); err != nil {
		t.Fatal(err)
	}
	fmt.Println(b)
}

func TestDuration(t *testing.T) {
	var d Duration
	if err := json.Unmarshal([]byte(`"1h1m1s1ms1us1ns"`), &d); err != nil {
		t.Fatal(err)
	}
	fmt.Println(d)
}

func TestGetIndianZeroUTC(_ *testing.T) {
	//t := time.Now()

	t, err := time.Parse(time.RFC3339, "2006-01-02T02:00:00+05:30")
	if err != nil {
		panic(err)
	}

	fmt.Println(GetIndianZeroUTC(t).Equal(GetZoneZeroUTC(t)))
}

//自定义类型，如何让标签有效
func TestValiDuration(t *testing.T) {
	va := validator.New()
	va.RegisterCustomTypeFunc(func(field reflect.Value) interface{} {
		d, ok := field.Interface().(Duration)
		fmt.Println(d, ok)
		return d.Duration
	}, Duration{})

	var s struct {
		D Duration `validate:"min=2"`
	}
	s.D.Duration = 1
	fmt.Println(va.Struct(s))
}
