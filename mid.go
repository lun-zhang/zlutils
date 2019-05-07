package zlutils

import (
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"strconv"
)

type AdminOperator struct {
	//请求中获取form，响应中写入json，数据库存gorm
	OperatorUid  string `gorm:"column:operator_uid" json:"operator_uid" form:"user_id"`
	OperatorName string `gorm:"column:operator_name" json:"operator_name" form:"username"`
}

const (
	KeyAdminOperator = "_key_admin_operator"
	KeyUser          = "_key_user"
)

func MidAdminOperator() gin.HandlerFunc {
	return func(c *gin.Context) {
		operator, err := func(c *gin.Context) (operator AdminOperator, err error) {
			if err = c.ShouldBindQuery(&operator); err != nil {
				return
			}
			if operator.OperatorName == "" || operator.OperatorUid == "" {
				err = fmt.Errorf("username:%s or user_id:%s empty",
					operator.OperatorName,
					operator.OperatorUid)
				return
			}
			return
		}(c)
		if err != nil {
			c.JSON(RespErrorQueryParams(fmt.Sprintf("invalid operator, err:%s", err.Error())))
			c.Abort()
			return
		}
		c.Set(KeyAdminOperator, operator)
	}
}

type User struct {
	UserId         string
	DeviceId       string
	ProductId      int
	AcceptLanguage string
}

func MidUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		var user User
		header := c.Request.Header
		user = User{
			UserId:         header.Get("User-Id"),
			DeviceId:       header.Get("Device-Id"),
			AcceptLanguage: header.Get("Accept-Language"),
		}
		user.ProductId, _ = strconv.Atoi(header.Get("Product-Id"))
		if user.UserId == "" && user.DeviceId == "" {
			c.JSON(RespErrorHeaderParams("invalid user, User-Id and Device-Id are empty"))
			c.Abort()
			return
		}
		c.Set(KeyUser, user)
	}
}

//NOTE: 如果想要上线后(level=info)想要某些接口打印日志，则增加一个类似于LogInfoWriter的struct
type LogDebugWriter struct {
	gin.ResponseWriter
}

func (w LogDebugWriter) Write(b []byte) (n int, err error) {
	logrus.WithFields(logrus.Fields{
		"stack":         nil,
		"response_body": string(b),
	}).Debug()
	return w.ResponseWriter.Write(b)
}

//NOTE: 上线后的接口不该使用这个中间件
//TODO: 以后增加level入参
func MidLogReqResp() gin.HandlerFunc {
	return func(c *gin.Context) {
		buf := new(bytes.Buffer)
		buf.ReadFrom(c.Request.Body)
		reqBody := buf.Bytes()
		logrus.WithFields(logrus.Fields{
			//TODO: 完善字段
			"path":         c.Request.URL.Path,
			"method":       c.Request.Method,
			"header":       c.Request.Header,
			"request_body": string(reqBody),
			"stack":        nil,
		}).Debug()
		c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(reqBody)) //拿出来再放回去
		c.Writer = LogDebugWriter{c.Writer}
		c.Next()
	}
}
