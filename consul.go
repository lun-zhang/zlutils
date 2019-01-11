package zlutils

import (
	"fmt"
	"math/rand"
	"sync"

	"encoding/json"
	"github.com/alecthomas/log4go"
	consulApi "github.com/hashicorp/consul/api"
	consulWatch "github.com/hashicorp/consul/watch"
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

func getSingle(key string) (value string) {
	key = fmt.Sprintf("%s/%s", Prefix, key)
	pair, _, err := KV.Get(key, nil)
	if err != nil {
		err = fmt.Errorf("key:%s, err:%+v", key, err)
		log4go.Error("%+v", err)
		panic(err)
	}
	if pair == nil {
		err = fmt.Errorf("consul has't key: %s", key)
		log4go.Error("%+v", err)
		panic(err)
	}
	value = string(pair.Value)
	return
}

func GetSingle(key string, i interface{}) {
	value := getSingle(key)
	if err := json.Unmarshal([]byte(value), i); err != nil {
		log4go.Error("consul:%s err:%+i", key, err)
		panic(err)
	}
	log4go.Info("consul:%s:%+v", key, reflect.ValueOf(i).Elem())
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
		panic(err)
	}

	KV = consulClient.KV()
	Catalog = consulClient.Catalog()

	fmt.Println("consulClient:", consulClient)
}
