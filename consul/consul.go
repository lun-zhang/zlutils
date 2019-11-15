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
	entry := logrus.WithField("key", key)
	pair, _, err := KV.Get(fmt.Sprintf("%s/%s", Prefix, key), nil)
	if err != nil {
		entry.WithError(err).Panic()
	}
	if pair == nil {
		entry.Panic("consul has't key")
	}
	//entry = entry.WithField("bs", string(value))
	//entry.Info("consul get bs ok")
	return pair.Value
}

var kv = map[string]reflect.Value{}

func getJson(key string, i interface{}, vaPtr *va) {
	t := reflect.TypeOf(i)
	entry := logrus.WithFields(logrus.Fields{
		"key":  key,
		"type": t.String(),
	})

	switch t.Kind() {
	case reflect.Ptr:
		value := GetValue(key)
		entry = entry.WithField("bs", string(value))
		if err := json.Unmarshal(value, i); err != nil {
			entry.WithError(err).Panic("consul value invalid")
		}
		entry = entry.WithField("value", i)
		if err := valiVa(vaPtr, i); err != nil {
			entry.WithError(err).Panicf("vali failed")
		}
		entry.Info("consul get value ok")
		kv[key] = reflect.ValueOf(i)
		return
	case reflect.Func:
		if t.NumIn() != 1 {
			entry.Panicf("numIn:%d != 1", t.NumIn())
		}
		v := reflect.ValueOf(i)

		in0Type := t.In(0)
		in0Ptr := reflect.New(in0Type).Interface()
		getJson(key, in0Ptr, vaPtr)
		v.Call([]reflect.Value{reflect.ValueOf(in0Ptr).Elem()})
		return
	default:
		entry.Panicf("invalid value kind:%s", t.Kind())
	}
}

//如果对值不关心，只想要用这个值去执行一个函数，例如用于初始化日志，那么第二个参数就传入有一个入参的函数吧
func GetJson(key string, i interface{}) {
	getJson(key, i, nil)
}

type va struct {
	t   int
	tag string
}

const (
	valiStruct = 1
	valiVar    = 2
)

func ValiStruct() va {
	return va{t: valiStruct}
}
func ValiVar(tag string) va {
	return va{
		t:   valiVar,
		tag: tag,
	}
}
func (m va) GetJson(key string, i interface{}) {
	getJson(key, i, &m)
}

func valiVa(vaPtr *va, i interface{}) error {
	if vaPtr == nil {
		return nil
	}
	switch vaPtr.t {
	case valiStruct:
		return vali.Struct(i)
	case valiVar:
		return vali.Var(i, vaPtr.tag)
	default:
		return fmt.Errorf("invalid m.t:%d", vaPtr.t)
	}
}

var vali = validator.New()

func (m va) WatchJson(key string, ptr interface{}, handler func()) {
	watchJson(key, ptr, handler, &m)
}

func WatchJson(key string, ptr interface{}, handler func()) {
	watchJson(key, ptr, handler, nil)
}

func watchJson(key string, ptr interface{}, handler func(), vaPtr *va) {
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
				entry.Errorf("panic: %v", r)
			}
		}()
		if kv, ok := raw.(*api.KVPair); ok && kv != nil {
			value = kv.Value
			entry = entry.WithField("bs", string(value))
			mysql.SetZero(ptr) //没有出现的字段不会被json.Unmarshal设置，因此这里先置零
			if err := json.Unmarshal(value, ptr); err != nil {
				entry.WithError(err).Errorf("consul watch unmarshal json failed")
				return
			}
			entry = entry.WithField("value", ptr)
			if err := valiVa(vaPtr, ptr); err != nil {
				entry.WithError(err).Error("vali failed")
				return
			}
			entry.Info("consul watch value ok")
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

func (m va) WatchJsonVarious(key string, i interface{}) {
	watchJsonVarious(key, i, &m)
}

//只关心修改后函数的执行
//consul监控key对应的value的变化，然后调用函数handler(value)
func WatchJsonVarious(key string, i interface{}) {
	watchJsonVarious(key, i, nil)
}

func watchJsonVarious(key string, i interface{}, vaPtr *va) {
	t := reflect.TypeOf(i)
	entry := logrus.WithFields(logrus.Fields{
		"key":  key,
		"type": t.String(),
	})
	switch t.Kind() {
	case reflect.Ptr:
		watchJson(key, i, nil, vaPtr)
	case reflect.Func:
		if t.NumIn() != 1 {
			entry.Panicf("numIn:%d != 1", t.NumIn())
		}
		v := reflect.ValueOf(i)

		in0Type := t.In(0)
		in0Ptr := reflect.New(in0Type).Interface() //为了避免再写一遍WatchJson而采用的偷懒做法
		watchJson(key, in0Ptr, func() {
			v.Call([]reflect.Value{reflect.ValueOf(in0Ptr).Elem()})
		}, vaPtr)
	}
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
