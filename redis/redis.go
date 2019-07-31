package redis

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"gopkg.in/redis.v5"
	"reflect"
	"time"
	"zlutils/caller"
)

//NOTE: 只能用于初始化时候，失败则fatal
//redis基本不会是性能瓶颈，所以不放xray
func New(url string) (client *Client) {
	redisOpt, err := redis.ParseURL(url)
	if err != nil {
		logrus.WithError(err).Fatalf("redis connect failed")
	}
	client = &Client{redis.NewClient(redisOpt)}
	//NOTE: pipeline没法用这个打日志
	client.WrapProcess(func(oldProcess func(cmd redis.Cmder) error) func(cmd redis.Cmder) error {
		return func(cmd redis.Cmder) error {
			begin := time.Now()
			err := oldProcess(cmd)
			end := time.Now()
			logrus.WithFields(logrus.Fields{
				"redis-cmd": cmd.String(),
				"duration":  end.Sub(begin).String(),
				"source":    caller.Caller(5), //FIXME: 这个数不太准
				"stack":     nil,
			}).Debug()
			return err
		}
	})
	return
}

type Client struct {
	*redis.Client
}

//TODO: 要加ctx的话就用WithContext
func (client *Client) SetJson(key string, value interface{}, expiration time.Duration) (err error) {
	entry := logrus.WithFields(logrus.Fields{
		"key":        key,
		"value":      value,
		"expiration": expiration,
	})
	bs, err := json.Marshal(value)
	if err != nil {
		entry.WithError(err).Error()
		return
	}
	cmd := client.Set(key, string(bs), expiration)
	entry = entry.WithField("cmd", cmd.String())
	if err = cmd.Err(); err != nil {
		entry.WithError(err).Error()
		return
	}
	return
}

func (client *Client) GetJson(key string, value interface{}) (err error) {
	entry := logrus.WithField("key", key)
	cmd := client.Get(key)
	entry = entry.WithField("cmd", cmd.String())
	if err = cmd.Err(); err != nil {
		if err != redis.Nil { //NOTE: nil不打err日志
			entry.WithError(err).Error()
		}
		return
	}
	if err = json.Unmarshal([]byte(cmd.Val()), value); err != nil {
		entry.WithError(err).Error()
		return
	}
	return
}

//存成map，方便不存储nil
func (client *Client) MGetJsonMap(keys []string, mapPtr interface{}) (err error) {
	rv := reflect.ValueOf(mapPtr)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		err = fmt.Errorf("type %s isn't ptr or is nil", reflect.TypeOf(mapPtr))
		logrus.WithError(err).Error()
		return
	}
	mp := rv.Elem()
	if mp.Kind() != reflect.Map {
		err = fmt.Errorf("kind %s isn't map", mp.Kind())
		logrus.WithError(err).Error()
		return
	}
	valueType := mp.Type().Elem() //value类型
	keyType := mp.Type().Key()    //key类型
	if keyType.Kind() != reflect.String {
		err = fmt.Errorf("key kind %s isn't string", keyType.Kind())
		logrus.WithError(err).Error()
		return
	}

	entry := logrus.WithField("keys", keys)
	cmds := client.MGet(keys...)
	entry = entry.WithField("cmds", cmds.String())
	if err = cmds.Err(); err != nil {
		entry.WithError(err).Error()
		return
	}
	mp.Set(reflect.MakeMap(mp.Type()))
	for i, val := range cmds.Val() {
		if val == nil {
			continue
		}
		newValuePtr := reflect.New(valueType).Interface()
		if err = json.Unmarshal([]byte(val.(string)), newValuePtr); err != nil {
			entry.WithError(err).Error()
			return
		}
		mp.SetMapIndex(reflect.ValueOf(keys[i]), reflect.ValueOf(newValuePtr).Elem())
	}
	return
}

//批量设置为相同的过期时间
func (client *Client) MultiSetJson(mp map[string]interface{}, expiration time.Duration) (err error) {
	entry := logrus.WithField("mp", mp)

	if len(mp) == 0 {
		return
	}

	pipe := client.Pipeline()
	defer pipe.Close()

	for k, v := range mp {
		bs, err := json.Marshal(v)
		if err != nil {
			entry.WithError(err).Error()
			return err
		}
		pipe.Set(k, string(bs), expiration)
	}

	cmds, err := pipe.Exec()

	var cmdss []string
	for _, cmd := range cmds {
		cmdss = append(cmdss, cmd.String())
		entry = entry.WithField("cmds", cmdss)
	}
	entry.Debug()
	if err != nil {
		entry.WithError(err).Error()
		return
	}
	return
}
