package consul

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
	"github.com/sirupsen/logrus"
	"net/http"
	"reflect"
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
		logrus.WithError(err).WithField("key", key).Fatal()
	}
	if pair == nil {
		err = fmt.Errorf("consul has't key")
		logrus.WithError(err).WithField("key", key).Fatal()
	}
	return pair.Value
}

var kv = map[string]reflect.Value{}

func GetJson(key string, i interface{}) {
	value := GetValue(key)
	if err := json.Unmarshal(value, i); err != nil {
		logrus.WithError(err).WithField(key, string(value)).Fatal("consul value invalid")
	}
	logrus.WithField(key, fmt.Sprintf("%+v", reflect.ValueOf(i).Elem())).Info("consul value ok")
	kv[key] = reflect.ValueOf(i)
}

func WatchJson(key string, i interface{}, handler func()) {
	plan, err := watch.Parse(map[string]interface{}{
		"type": "key",
		"key":  fmt.Sprintf("%s/%s", Prefix, key),
	})
	if err != nil {
		logrus.WithError(err).Fatal("consul watch parse failed")
	}
	plan.Handler = func(idx uint64, raw interface{}) {
		if kv, ok := raw.(*api.KVPair); ok && kv != nil {
			if err := json.Unmarshal(kv.Value, i); err != nil {
				logrus.WithError(err).WithField(key, string(kv.Value)).Errorf("consul watch unmarshal json failed")
				return
			}
			handler() //发生修改后回调
		} else {
			logrus.WithFields(logrus.Fields{
				"key": key,
				"idx": idx,
				"raw": raw,
			}).Errorf("consul watch invalid raw")
		}
	}
	go plan.Run(Address)
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
		entry.WithError(err).Fatal("consul connect failed")
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
