package time

import (
	"encoding/json"
	"fmt"
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

func TestGetIndianZeroUTC(t *testing.T) {
	fmt.Println(GetIndianZeroUTC(time.Now()))
}
