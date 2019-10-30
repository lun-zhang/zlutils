package misc

import "testing"

func TestIsNil(t *testing.T) {
	a := 1
	tests := []struct {
		i     interface{}
		isNil bool
	}{
		{nil, true},
		{map[int]string(nil), true},
		{struct{}{}, false},
		{&a, false},
		{1, false},
		{(*int)(nil), true},
	}

	for i, test := range tests {
		get := IsNil(test.i)
		want := test.isNil
		if get != want {
			t.Errorf("%d failed get:%v, want:%v", i, get, want)
		} else {
			t.Logf("%d ok", i)
		}
	}
}
