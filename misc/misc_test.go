package misc

import (
	"fmt"
	"testing"
	"time"
)

func TestMin(t *testing.T) {
	for i, test := range []struct {
		a int
		b []int
		t int
	}{
		{1, []int{2, 3}, 1},
		{-1, []int{-2, -3}, -3},
	} {
		if o := MinInt(test.a, test.b...); o != test.t {
			t.Errorf("%d %d != %d", i, test.t, o)
		} else {
			t.Logf("%d ok", i)
		}
	}
}

func TestMax(t *testing.T) {
	for i, test := range []struct {
		a int8
		b []int8
		t int8
	}{
		{1, []int8{2, 3}, 3},
		{-1, []int8{-2, -3}, -1},
	} {
		if o := MaxInt8(test.a, test.b...); o != test.t {
			t.Errorf("%d %d != %d", i, test.t, o)
		} else {
			t.Logf("%d ok", i)
		}
	}
}

func TestMinDuration(t *testing.T) {
	fmt.Println(MinDuration(time.Second, time.Minute))
}
