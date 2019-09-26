package session

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"strconv"
	"zlutils/code"
	"zlutils/meta"
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

func MidOperator() gin.HandlerFunc {
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
			code.Send(c, nil, code.ClientErrQuery.WithErrorf("invalid operator, err:%s", err.Error()))
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
		user.UserIdentity = user.UserId //VClip用UserId，因为DeviceId会发生变化
	default: //默认按videobuddy来
		if user.UserId != "" {
			user.UserIdentity = user.UserId
		} else {
			user.UserIdentity = user.DeviceId
		}
	}
}

//NOTE: 这两个接口如果调用失败则panic，使用了对应中间件后一定成功
func GetOperator(c *gin.Context) Operator {
	return Meta(c.Keys).GetOperator()
}

func GetUser(c *gin.Context) User {
	return Meta(c.Keys).GetUser()
}

type Meta meta.Meta

func (m Meta) Meta() meta.Meta {
	return meta.Meta(m)
}
func (m Meta) GetOperator() Operator {
	return meta.Meta(m).MustGet(KeyOperator).(Operator)
}

func (m Meta) GetUser() User {
	return meta.Meta(m).MustGet(KeyUser).(User)
}

const (
	ProductIdVideoBuddy = 39
	ProductIdVClip      = 45
)

//FIXME 感觉这个不是公用的，不改放这里
func MidUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		var err error
		header := c.Request.Header
		vc, _ := strconv.Atoi(header.Get("Version-Code"))
		user := User{
			//FIXME 从token中获得用户信息，兼容新token
			UserId:         header.Get("User-Id"),
			DeviceId:       header.Get("Device-Id"),
			AcceptLanguage: header.Get("Accept-Language"),
			VersionCode:    vc,
		}
		user.ProductId, _ = strconv.Atoi(header.Get("Product-Id"))
		switch user.ProductId {
		case ProductIdVClip:
			if user.UserId == "" {
				err = fmt.Errorf("User-Id is empty") //vclip以UserId为主，所以必须有
			}
		case ProductIdVideoBuddy:
			if user.UserId == "" && user.DeviceId == "" {
				err = fmt.Errorf("User-Id and Device-Id are empty")
			}
		default:
			//TODO 后续增加新类型
		}
		if err != nil {
			code.Send(c, nil, code.ClientErrHeader.WithErrorf("invalid user, %s", err.Error()))
			c.Abort()
		} else {
			user.refreshUserIdentity()
			c.Set(KeyUser, user)
		}
	}
}

//调用此中间件前必须调用MidUser中间件，否则panic，应当在测试时候排查
//或者自行设置user，但须保证UserId和DeviceId不同时为空
//与MidUser分离的目的是，如果不满意MidUser的实现，可以自行实现，并且还能用这个绑定中间件
func MidBindUserVideoBuddy(bind request.Config) gin.HandlerFunc {
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
			code.Send(c, nil, code.ServerErrRpc.WithErrorf("bind user failed"))
			c.Abort()
			return
		}
		user.UserId = resp.Data.UserId
		user.DeviceId = resp.Data.DeviceId
		user.refreshUserIdentity()
		c.Set(KeyUser, user)
	}
}
