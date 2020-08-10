package consul

import (
	"encoding/json"
	"fmt"
	"github.com/go-playground/validator"
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"reflect"
	"zlutils/mysql"
)

var (
	Address string
	Prefix  string
	KV      *api.KV // KV is used to manipulate the K/V API
	Client  *api.Client
)

//自定义前缀，这样就可以复用配置了
func WithPrefix(prefix string) Consul {
	return Consul{prefixPtr: &prefix}
}

func (m Consul) WithPrefix(prefix string) Consul {
	m.prefixPtr = &prefix
	return m
}

func GetValue(key string) (value []byte) {
	return getValue(key, defaultConsul)
}
func getValue(key string, lo Consul) (value []byte) {
	entry := logrus.WithField("key", key)

	prefix := Prefix
	if lo.prefixPtr != nil {
		prefix = *lo.prefixPtr
	}

	pair, _, err := KV.Get(fmt.Sprintf("%s/%s", prefix, key), nil)
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

func (m Consul) GetValue(key string) (value []byte) {
	return getValue(key, m)
}

var kv = map[string]reflect.Value{}

type Unmarshal func(data []byte, v interface{}) error

func getJson(key string, i interface{}, lo Consul, unmarshal Unmarshal) {
	t := reflect.TypeOf(i)
	entry := logrus.WithFields(logrus.Fields{
		"key":  key,
		"type": t.String(),
	})

	switch t.Kind() {
	case reflect.Ptr:
		value := getValue(key, lo)
		entry = entry.WithField("bs", string(value))
		if err := unmarshal(value, i); err != nil {
			entry.WithError(err).Panic("consul value invalid")
		}
		entry = entry.WithField("value", i)
		if err := valiVa(lo, i); err != nil {
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
		getJson(key, in0Ptr, lo, unmarshal)
		v.Call([]reflect.Value{reflect.ValueOf(in0Ptr).Elem()})
		return
	default:
		entry.Panicf("invalid value kind:%s", t.Kind())
	}
}

//如果对值不关心，只想要用这个值去执行一个函数，例如用于初始化日志，那么第二个参数就传入有一个入参的函数吧
func GetJson(key string, i interface{}) {
	getJson(key, i, defaultConsul, json.Unmarshal)
}

func GetYaml(key string, i interface{}) {
	getJson(key, i, defaultConsul, yaml.Unmarshal)
}

//成员用指针，不为nil时候才使用对应功能
type Consul struct {
	valiTypePtr *int
	tag         string
	prefixPtr   *string
}

var defaultConsul Consul //默认的

const (
	valiStruct = 1
	valiVar    = 2
)

func newInt(i int) *int {
	return &i
}

func (m Consul) ValiStruct() Consul {
	m.valiTypePtr = newInt(valiStruct)
	return m
}

func (m Consul) ValiVar(tag string) Consul {
	m.valiTypePtr = newInt(valiVar)
	m.tag = tag
	return m
}

func ValiStruct() Consul {
	return Consul{valiTypePtr: newInt(valiStruct)}
}
func ValiVar(tag string) Consul {
	return Consul{
		valiTypePtr: newInt(valiVar),
		tag:         tag,
	}
}
func (m Consul) GetJson(key string, i interface{}) {
	getJson(key, i, m, json.Unmarshal)
}

func (m Consul) GetYaml(key string, i interface{}) {
	getJson(key, i, m, yaml.Unmarshal)
}

func valiVa(lo Consul, i interface{}) error {
	if lo.valiTypePtr == nil {
		return nil
	}
	switch *lo.valiTypePtr {
	case valiStruct:
		return vali.Struct(i)
	case valiVar:
		return vali.Var(i, lo.tag)
	default:
		return fmt.Errorf("invalid m.valiType:%d", lo.valiTypePtr)
	}
}

var vali = validator.New()

func (m Consul) WatchJson(key string, ptr interface{}, handler func()) {
	watchJson(key, ptr, handler, m, json.Unmarshal)
}
func (m Consul) WatchYaml(key string, ptr interface{}, handler func()) {
	watchJson(key, ptr, handler, m, yaml.Unmarshal)
}

func WatchJson(key string, ptr interface{}, handler func()) {
	watchJson(key, ptr, handler, defaultConsul, json.Unmarshal)
}

func WatchYaml(key string, ptr interface{}, handler func()) {
	watchJson(key, ptr, handler, defaultConsul, yaml.Unmarshal)
}

func watchJson(key string, ptr interface{}, handler func(), lo Consul, unmarshal Unmarshal) {
	prefix := Prefix
	if lo.prefixPtr != nil {
		prefix = *lo.prefixPtr
	}

	plan, err := watch.Parse(map[string]interface{}{
		"type": "key",
		"key":  fmt.Sprintf("%s/%s", prefix, key),
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
			if err := unmarshal(value, ptr); err != nil {
				entry.WithError(err).Errorf("consul watch unmarshal json failed")
				return
			}
			entry = entry.WithField("value", ptr)
			if err := valiVa(lo, ptr); err != nil {
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

func (m Consul) WatchJsonVarious(key string, i interface{}) {
	watchJsonVarious(key, i, m, json.Unmarshal)
}

func (m Consul) WatchYamlVarious(key string, i interface{}) {
	watchJsonVarious(key, i, m, yaml.Unmarshal)
}

//只关心修改后函数的执行
//consul监控key对应的value的变化，然后调用函数handler(value)
func WatchJsonVarious(key string, i interface{}) {
	watchJsonVarious(key, i, defaultConsul, json.Unmarshal)
}

func WatchYamlVarious(key string, i interface{}) {
	watchJsonVarious(key, i, defaultConsul, yaml.Unmarshal)
}

func watchJsonVarious(key string, i interface{}, lo Consul, unmarshal Unmarshal) {
	t := reflect.TypeOf(i)
	entry := logrus.WithFields(logrus.Fields{
		"key":  key,
		"type": t.String(),
	})
	switch t.Kind() {
	case reflect.Ptr:
		watchJson(key, i, nil, lo, unmarshal)
	case reflect.Func:
		if t.NumIn() != 1 {
			entry.Panicf("numIn:%d != 1", t.NumIn())
		}
		v := reflect.ValueOf(i)

		in0Type := t.In(0)
		in0Ptr := reflect.New(in0Type).Interface() //为了避免再写一遍WatchJson而采用的偷懒做法
		watchJson(key, in0Ptr, func() {
			v.Call([]reflect.Value{reflect.ValueOf(in0Ptr).Elem()})
		}, lo, unmarshal)
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
	Client = consulClient
	KV = consulClient.KV()
	entry.Info("consul connect ok")
}
