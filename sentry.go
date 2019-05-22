package zlutils

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http/httputil"
	"strconv"
)

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				endpoint := fmt.Sprintf("%s-%s", c.Request.URL.Path, c.Request.Method)
				httpRequest, _ := httputil.DumpRequest(c.Request, true)
				logrus.WithFields(logrus.Fields{
					"httpRequest": string(httpRequest),
					"stack":       GetStack(3),
					"recover":     rec,
					"endpoint":    endpoint,
				}).Error()

				CodeSend(c, nil, CodeServerMidPaincErr.WithError(fmt.Errorf("panic recover: %s", rec)))
				ServerErrorCounter.WithLabelValues(endpoint, strconv.Itoa(c.Value(KeyRet).(int))).Inc()
				c.Abort()
			}
		}()
		c.Next()
	}
}
