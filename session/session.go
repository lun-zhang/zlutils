package session

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"zlutils/request"
)

type Operator struct {
	//请求中获取form，响应中写入json，数据库存gorm
	OperatorUid  string `gorm:"column:operator_uid" json:"operator_uid" form:"user_id"`
	OperatorName string `gorm:"column:operator_name" json:"operator_name" form:"username"`
}

const (
	KeyOperator = "_key_operator"
	KeyUser     = "_key_user"
)

type sendFunc func(c *gin.Context, data interface{}, err error)

func MidOperator(sendClientErrQuery sendFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		operator, err := func(c *gin.Context) (operator Operator, err error) {
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
			if sendClientErrQuery != nil {
				sendClientErrQuery(c, nil, fmt.Errorf("invalid operator, err:%s", err.Error()))
			} else {
				c.JSON(http.StatusBadRequest, nil)
			}
			c.Abort()
			return
		}
		c.Set(KeyOperator, operator)
	}
}

type User struct {
	UserIdentity   string `json:"user_identity"` //NOTE: 不同产品的用户唯一标志不同
	UserId         string `json:"user_id"`
	DeviceId       string `json:"device_id"`
	ProductId      int    `json:"product_id"`
	AcceptLanguage string `json:"accept_language"`
	VersionCode    int    `json:"version_code"`
}

//发生变化后重设UserIdentity
func (user *User) refreshUserIdentity() {
	switch user.ProductId {
	case ProductIdVClip:
		user.UserIdentity = user.DeviceId //NOTE: vclip以device_id为唯一身份
	case ProductIdVideoBuddy:
		if user.UserId != "" {
			user.UserIdentity = user.UserId //NOTE: videoBuddy优先user-id为唯一标志
		} else {
			user.UserIdentity = user.DeviceId
		}
	}
}

//NOTE: 这两个接口如果调用失败则panic，使用了对应中间件后一定成功
func GetOperator(c *gin.Context) Operator {
	return c.Value(KeyOperator).(Operator)
}

func GetUser(c *gin.Context) User {
	return c.Value(KeyUser).(User)
}

const (
	ProductIdVideoBuddy = 39
	ProductIdVClip      = 45
)

//FIXME 感觉这个不是公用的，不改放这里
func MidUser(sendClientErrHeader sendFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := func(c *gin.Context) (user User, err error) {
			header := c.Request.Header
			vc, _ := strconv.Atoi(header.Get("Version-Code"))
			user = User{
				//FIXME 从token中获得用户信息，兼容新token
				UserId:         header.Get("User-Id"),
				DeviceId:       header.Get("Device-Id"),
				AcceptLanguage: header.Get("Accept-Language"),
				VersionCode:    vc,
			}
			user.ProductId, _ = strconv.Atoi(header.Get("Product-Id"))
			switch user.ProductId {
			case ProductIdVClip:
				if user.DeviceId == "" { //NOTE: vclip一定有Device-Id，可能没有User-Id
					err = fmt.Errorf("Device-Id is empty")
					return
				}
			case ProductIdVideoBuddy:
				if user.UserId == "" && user.DeviceId == "" {
					err = fmt.Errorf("User-Id and Device-Id are empty")
					return
				}
			default:
				//TODO 后续增加新类型
			}
			user.refreshUserIdentity()
			return
		}(c)
		if err != nil {
			if sendClientErrHeader != nil {
				sendClientErrHeader(c, nil, fmt.Errorf("invalid user, %s", err.Error()))
			} else {
				c.JSON(http.StatusBadRequest, nil)
			}
			c.Abort()
			return
		}
		c.Set(KeyUser, user)
	}
}

//调用此中间件前必须调用MidUser中间件，否则panic，应当在测试时候排查
//或者自行设置user，但须保证UserId和DeviceId不同时为空
//与MidUser分离的目的是，如果不满意MidUser的实现，可以自行实现，并且还能用这个绑定中间件
func MidBindUserVideoBuddy(bind request.Config, sendServerErrRpc sendFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := GetUser(c)
		if user.ProductId != ProductIdVideoBuddy {
			return //只绑定vb的
		}
		var resp struct {
			request.RespRet
			Data struct {
				WasBound bool   `json:"was_bound"`
				UserId   string `json:"user_id"`
				DeviceId string `json:"device_id"`
			} `json:"data"`
		}
		req := request.Request{
			Config: bind,
		}
		if user.UserId != "" {
			req.AddQuery("user_id", user.UserId)
		}
		if user.DeviceId != "" {
			req.AddQuery("device_id", user.DeviceId)
		}
		if err := req.Do(c.Request.Context(), &resp); err != nil {
			if sendServerErrRpc != nil {
				sendServerErrRpc(c, nil, fmt.Errorf("bind user failed"))
			} else {
				c.JSON(http.StatusInternalServerError, nil)
			}
			c.Abort()
			return
		}
		user.UserId = resp.Data.UserId
		user.DeviceId = resp.Data.DeviceId
		user.refreshUserIdentity()
		c.Set(KeyUser, user)
	}
}
