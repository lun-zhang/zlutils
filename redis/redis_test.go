package redis

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"testing"
	"time"
	"zlutils/logger"
)

var ctx = context.Background()

func TestClient_GetJson(t *testing.T) {
	logger.Init(logger.Config{Level: logrus.DebugLevel})
	client := New("redis://localhost:6379")
	type C struct {
		D bool
	}
	type A struct {
		A int    `json:"a"`
		B string `json:"b"`
		C C      `json:"c"`
	}

	if err := client.SetJson(ctx, "a", A{
		A: 1,
		B: "b",
		C: C{
			D: true,
		},
	}, time.Hour); err != nil {
		t.Fatal(err)
	}

	var a A
	if err := client.GetJson(ctx, "a", &a); err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%#v", a)
}

func TestClient_MGetJsonMap(t *testing.T) {
	logger.Init(logger.Config{Level: logrus.DebugLevel})
	client := New("redis://localhost:6379")
	if err := client.MultiSetJson(ctx, map[string]interface{}{
		"b": 1,
		"c": "b",
	}, time.Hour); err != nil {
		t.Fatal(err)
	}

	var mp map[string]interface{}
	if err := client.MGetJsonMap(ctx, []string{"b", "c", "d"}, &mp); err != nil {
		t.Fatal(err)
	}
	fmt.Println(mp)
}
