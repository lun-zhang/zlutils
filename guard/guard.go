package guard

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

func Mid(send func(c *gin.Context, data interface{}, err error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				if send != nil {
					send(c, nil, fmt.Errorf("panic: %+v", rec)) //用户自定义处理
				} else {
					c.JSON(http.StatusInternalServerError, nil) //默认返回500
				}
			}
		}()
		c.Next()
	}
}
