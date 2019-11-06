package consul

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator"
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
	"github.com/sirupsen/logrus"
	"net/http"
	"reflect"
	"zlutils/mysql"
)

var (
	Address string
	Prefix  string
	KV      *api.KV // KV is used to manipulate the K/V API
	Catalog *api.Catalog
)

func GetValue(key string) (value []byte) {
	pair, _, err := KV.Get(fmt.Sprintf("%s/%s", Prefix, key), nil)
	if err != nil {
		logrus.WithError(err).WithField("key", key).Panic()
	}
	if pair == nil {
		err = fmt.Errorf("consul has't key")
		logrus.WithError(err).WithField("key", key).Panic()
	}
	return pair.Value
}

var kv = map[string]reflect.Value{}

func GetJson(key string, ptr interface{}) {
	value := GetValue(key)
	if err := json.Unmarshal(value, ptr); err != nil {
		logrus.WithError(err).WithField(key, string(value)).Panic("consul value invalid")
	}
	logrus.WithField(key, fmt.Sprintf("%+v", reflect.ValueOf(ptr).Elem())).Info("consul value ok")
	kv[key] = reflect.ValueOf(ptr)
}

var vali = validator.New()

func GetJsonValiStruct(key string, ptr interface{}) {
	GetJson(key, ptr)
	if err := vali.Struct(ptr); err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"key": key,
			"ptr": ptr,
		}).Panic()
	}
}
func GetJsonValiVar(key string, ptr interface{}, tag string) {
	GetJson(key, ptr)
	if err := vali.Var(ptr, tag); err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"key": key,
			"ptr": ptr,
		}).Panic()
	}
}

func WatchJson(key string, ptr interface{}, handler func()) {
	plan, err := watch.Parse(map[string]interface{}{
		"type": "key",
		"key":  fmt.Sprintf("%s/%s", Prefix, key),
	})
	if err != nil {
		logrus.WithField("key", key).
			WithError(err).
			Panic("consul watch parse failed")
	}
	plan.Handler = func(idx uint64, raw interface{}) {
		entry := logrus.WithField("key", key)
		var value []byte
		defer func() {
			//避免对外部造成影响
			if r := recover(); r != nil {
				entry.WithField("value", string(value)).
					Errorf("panic: %v", r)
			}
		}()
		if kv, ok := raw.(*api.KVPair); ok && kv != nil {
			value = kv.Value
			mysql.SetZero(ptr) //没有出现的字段不会被json.Unmarshal设置，因此这里先置零
			if err := json.Unmarshal(kv.Value, ptr); err != nil {
				entry.WithError(err).
					WithField("value", string(kv.Value)).
					Errorf("consul watch unmarshal json failed")
				return
			}
			if handler != nil {
				handler() //启动时会起个线程执行一次，发生修改后回调
			}
		} else {
			entry.WithFields(logrus.Fields{
				"idx": idx,
				"raw": raw,
			}).Errorf("consul watch invalid raw")
		}
	}
	go plan.Run(Address)
}

//只关心修改后函数的执行
//consul监控key对应的value的变化，然后调用函数handler(value)
func WatchJsonHandler(key string, handler interface{}) {
	t := reflect.TypeOf(handler)
	entry := logrus.WithFields(logrus.Fields{
		"key":     key,
		"handler": t.String(),
	})
	if t.Kind() != reflect.Func {
		entry.Panicf("handler kind:%s is'nt func", t.Kind())
	}
	if t.NumIn() != 1 {
		entry.Panicf("numIn:%d != 1", t.NumIn())
	}
	v := reflect.ValueOf(handler)

	in0Type := t.In(0)
	in0Ptr := reflect.New(in0Type).Interface()
	WatchJson(key, in0Ptr, func() {
		v.Call([]reflect.Value{reflect.ValueOf(in0Ptr).Elem()})
	})
}

func Init(address string, prefix string) {
	entry := logrus.WithFields(logrus.Fields{
		"address": address,
		"prefix":  prefix,
	})
	Address = address
	Prefix = prefix
	consulClient, err := api.NewClient(&api.Config{Address: address})
	if err != nil {
		entry.WithError(err).Panic("consul connect failed")
	}
	KV = consulClient.KV()
	Catalog = consulClient.Catalog()
	entry.Info("consul connect ok")
}

func BindRouter(group *gin.RouterGroup) {
	config := group.Group("consul/kv")
	var ks []string
	for k, v := range kv {
		ks = append(ks, k)
		config.GET(k, func(v reflect.Value) gin.HandlerFunc {
			return func(c *gin.Context) {
				c.JSON(http.StatusOK, v.Interface())
			}
		}(v))
		config.PUT(k, func(v reflect.Value) gin.HandlerFunc {
			return func(c *gin.Context) {
				reqBodyPtr := reflect.New(v.Elem().Type()).Interface() //创建一个空的
				if err := c.ShouldBindJSON(reqBodyPtr); err != nil {
					c.JSON(http.StatusBadRequest, err.Error())
					return
				}
				v.Elem().Set(reflect.ValueOf(reqBodyPtr).Elem())
				c.JSON(http.StatusOK, v.Interface())
			}
		}(v))
	}
	config.GET("", func(c *gin.Context) {
		c.JSON(http.StatusOK, ks)
	})
}
