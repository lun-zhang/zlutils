package misc

import (
	"math/rand"
	"time"
)

//在[min(a,b),max(a,b)]中随机
func RandIntMinMax(a, b int) int {
	if a == b {
		return a
	}
	if a > b {
		a, b = b, a
	}
	return rand.Intn(b-a+1) + a
}

func RandInt64MinMax(a, b int64) int64 {
	if a == b {
		return a
	}
	if a > b {
		a, b = b, a
	}
	return rand.Int63n(b-a+1) + a
}

func RandDurationMinMax(a, b time.Duration) time.Duration {
	return time.Duration(RandInt64MinMax(int64(a), int64(b)))
}
