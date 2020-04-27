package session

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"zlutils/code"
	"zlutils/meta"
	"zlutils/request"
	"zlutils/time"
)

//请求中获取form，响应中写入json，数据库存gorm
type Operator struct {
	OperatorUid  string `gorm:"column:operator_uid" json:"operator_uid" form:"user_id"`    //操作者id
	OperatorName string `gorm:"column:operator_name" json:"operator_name" form:"username"` //操作者名字
}

//通常管理后台的表都有插入、更新时间
type OperatorWithTime struct {
	Operator
	UpdatedAt time.Time `json:"updated_at" gorm:"column:updated_at"` //更新时间
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at"` //创建时间
}

const (
	keyOperator = "_key_operator"
	keyUser     = "_key_user"
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
			logrus.WithContext(c.Request.Context()).WithError(err).Warn()
			code.Send(c, nil, code.ClientErrQuery.WithErrorf("invalid operator, err:%s", err.Error()))
			c.Abort()
			return
		}
		c.Set(keyOperator, operator)
	}
}

type User struct {
	UserIdentity   string `json:"user_identity" header:"-"`                          //NOTE: 不同产品的用户唯一标志不同
	UserId         string `json:"user_id" header:"User-Id"`                          //用户id
	DeviceId       string `json:"device_id" header:"Device-Id"`                      //设备id
	ProductId      int    `json:"product_id" header:"Product-Id" binding:"required"` //产品id
	AcceptLanguage string `json:"accept_language" header:"Accept-Language"`          //设备语言
	DeviceLanguage string `json:"device_language" header:"Device-Language"`          //app语言
	VersionCode    int    `json:"version_code" header:"Version-Code"`                //版本号
	SimCountry     string `json:"sim_country" header:"Sim-Country"`                  //sim卡所在地
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
	return meta.Meta(m).MustGet(keyOperator).(Operator)
}

func (m Meta) GetUser() User {
	return meta.Meta(m).MustGet(keyUser).(User)
}

const (
	ProductIdVideoBuddy = 39
	ProductIdVClip      = 45
	ProductIdManiGames  = 49
	ProductIdFunnyStar  = 50
)

type withoutValidate struct{}

//治标：有的服务例如share不需要一定有User-Id Device-Id
func (withoutValidate) MidUser() gin.HandlerFunc {
	return midUser(true)
}

//这样比MidUser加入参要不容易误用
func WithoutValidate() withoutValidate {
	return withoutValidate{}
}

//FIXME 感觉这个不是公用的，不改放这里
func MidUser() gin.HandlerFunc {
	return midUser(false)
}

func midUser(noValidate bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var err error
		var user User
		if err = c.ShouldBindHeader(&user); err != nil {
			code.Send(c, nil, code.ClientErrHeader.WithError(err))
			c.Abort()
			return
		}
		if !noValidate {
			switch user.ProductId {
			case ProductIdVClip:
				if user.UserId == "" || user.DeviceId == "" {
					err = fmt.Errorf("User-Id or Device-Id is empty") //vclip以UserId为主，所以必须有，加金币时需要Device-Id，所以两者都要有
				}
			case ProductIdVideoBuddy:
				if user.UserId == "" && user.DeviceId == "" {
					err = fmt.Errorf("User-Id and Device-Id are empty")
				}
			default:
				//TODO 后续增加新类型
			}
			if err != nil {
				logrus.WithContext(c.Request.Context()).WithError(err).Warn()
				code.Send(c, nil, code.ClientErrHeader.WithErrorf("invalid user, %s", err.Error()))
				c.Abort()
				return
			}
		}
		user.refreshUserIdentity()
		c.Set(keyUser, user)
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
		c.Set(keyUser, user)
	}
}
