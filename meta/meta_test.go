package meta

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"testing"
)

func TestMeta_Set(t *testing.T) {
	var m Meta
	m.Set("a", 1)
	fmt.Println(m)
}

func TestMeta_MustGet(t *testing.T) {
	var m0 Meta
	m0.Set("a", 1)
	fmt.Println(m0.MustGet("a"))
}

func TestMeta_MustGet2(t *testing.T) {
	var m Meta
	m.MustGet("a")
}

func TestGinKeys(t *testing.T) {
	c := &gin.Context{}
	c.Set("a", 1)
	var m Meta
	m = c.Keys
	fmt.Println(m.MustGet("a"))
}
