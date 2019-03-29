package zlutils

import (
	"fmt"
	"math/rand"
	"sync"

	"encoding/json"
	consulApi "github.com/hashicorp/consul/api"
	consulWatch "github.com/hashicorp/consul/watch"
	"github.com/sirupsen/logrus"
	"reflect"
)

var (
	Address string
	Prefix  string
	KV      *consulApi.KV // KV is used to manipulate the K/V API
	Catalog *consulApi.Catalog
)

type WatchedParam struct {
	value string
	lock  sync.RWMutex
}

func (v *WatchedParam) Get() string {
	v.lock.RLock()
	defer v.lock.RUnlock()
	return v.value
}

func (v *WatchedParam) Set(value string) {
	v.lock.Lock()
	defer v.lock.Unlock()
	v.value = value
}

func GetValue(key string) (value []byte) {
	key = fmt.Sprintf("%s/%s", Prefix, key)
	pair, _, err := KV.Get(key, nil)
	if err != nil {
		logrus.WithError(err).WithField("key", key).Fatal()
	}
	if pair == nil {
		err = fmt.Errorf("consul has't key")
		logrus.WithError(err).WithField("key", key).Fatal()
	}
	return pair.Value
}

func UnmarshalJson(key string, i interface{}) {
	value := GetValue(key)
	if err := json.Unmarshal(value, i); err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"key":   key,
			"value": string(value),
		}).Fatal("consul key invalid")
	}
	logrus.WithFields(logrus.Fields{
		"key":   key,
		"value": fmt.Sprintf("%+v", reflect.ValueOf(i).Elem()),
	}).Info()
}

func WatchSingle(key string, param *WatchedParam) {
	params := map[string]interface{}{
		"type": "key",
		"key":  fmt.Sprintf("%s/%s", Prefix, key),
	}
	plan, _ := consulWatch.Parse(params)
	plan.Handler = func(idx uint64, raw interface{}) {
		if raw == nil {
			return
		}

		v, ok := raw.(*consulApi.KVPair)
		if ok && v != nil {
			newValue := string(v.Value)
			param.Set(newValue)
		}
	}

	go plan.Run(Address)
}

func GetServiceAddress(serviceName string) string {
	pair, _, err := Catalog.Service(serviceName, "", nil)
	if err != nil {
		panic(err)
	}

	pairLength := len(pair)
	if pairLength == 0 {
		panic(fmt.Errorf("%s not exist", serviceName))
	}

	index := rand.Intn(pairLength)
	topService := pair[index]
	return fmt.Sprintf("%s:%d", topService.ServiceAddress, topService.ServicePort)
}

func InitConsul(address string, prefix string) {
	Address = address
	Prefix = prefix

	conConfig := consulApi.Config{Address: address}
	consulClient, err := consulApi.NewClient(&conConfig)
	if err != nil {
		logrus.WithError(err).Fatal()
	}

	KV = consulClient.KV()
	Catalog = consulClient.Catalog()

	logrus.WithField("consulClient", fmt.Sprintf("%v", consulClient)).Info()
}
