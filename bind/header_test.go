package bind

import (
	"fmt"
	"net/http"
	"testing"
)

func TestBindHeader(t *testing.T) {
	header := http.Header{}
	header.Add("S", "s")
	header.Add("I", "1")
	header.Add("J", "1")
	header.Add("f", "1.1")
	header.Add("a-b", "1") //被转成大写A-B

	var reqHeader struct {
		S  string  `header:"s"`
		I  int     `header:"-"` //忽略
		J  int     //没tag时候，名字为J
		F  float32 `header:"f"`
		AB int     `header:"A-B"`
		//No int     `header:"no" binding:"required"` //检验
	}
	reqHeader.S = "init" //NOTE: 发生错误时，不会被修改
	if err := ShouldBindHeader(header, &reqHeader); err != nil {
		fmt.Println(err) //如果失败，则reqHeader会被置零
	}
	fmt.Printf("%+v\n", reqHeader)
}

func TestBindHeaderAnonymous(t *testing.T) {
	header := http.Header{}
	header.Add("i", "1")
	type I struct {
		I int `header:"i" binding:"required"`
	}
	var reqHeader struct {
		I
	}
	if err := ShouldBindHeader(header, &reqHeader); err != nil {
		fmt.Println(err)
	}
	fmt.Printf("%+v\n", reqHeader)
}
