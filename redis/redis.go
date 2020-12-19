package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"gopkg.in/redis.v5"
	"reflect"
	"time"
	"zlutils/caller"
	"zlutils/guard"
)

//NOTE: 只能用于初始化时候，失败则panic
//redis基本不会是性能瓶颈，所以不放xray
func New(url string) (client *Client) {
	redisOpt, err := redis.ParseURL(url)
	if err != nil {
		logrus.WithError(err).Panic("redis connect failed")
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
func (client *Client) SetJson(ctx context.Context, key string, value interface{}, expiration time.Duration) (err error) {
	defer guard.BeforeCtx(&ctx)(&err)
	entry := logrus.WithContext(ctx).WithFields(logrus.Fields{
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

func (client *Client) GetJson(ctx context.Context, key string, value interface{}) (err error) {
	defer guard.BeforeCtx(&ctx)(&err)
	entry := logrus.WithContext(ctx).WithField("key", key)
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

func checkKeyFunc(funcType, bizKeyType reflect.Type) error {
	if kind := funcType.Kind(); kind != reflect.Func {
		return fmt.Errorf("keyFunc kind %s must be func", kind)
	}
	//检查入参
	if n := funcType.NumIn(); n != 1 {
		return fmt.Errorf("keyFunc num in %d must be 1", n)
	}
	in0Type := funcType.In(0)
	if in0Type != bizKeyType {
		return fmt.Errorf("keyFuncIn0Type %s must eq bizKeyType %s", in0Type, bizKeyType)
	}
	//检查出参
	if n := funcType.NumOut(); n != 1 {
		return fmt.Errorf("keyFunc num out %d must be 1", n)
	}
	if kind := funcType.Out(0).Kind(); kind != reflect.String {
		return fmt.Errorf("keyFuncOut0Kind %s must be string", kind)
	}
	return nil
}

func checkFillFunc(funcType, bizKeysType, outType reflect.Type) error {
	if kind := funcType.Kind(); kind != reflect.Func {
		return fmt.Errorf("fillFunc kind %s must be func", kind)
	}
	//检查入参
	if n := funcType.NumIn(); n != 2 {
		return fmt.Errorf("fillFunc num in %d must be 2", n)
	}

	in0Type := funcType.In(0)
	if _, ok := reflect.New(in0Type).Interface().(*context.Context); !ok {
		return fmt.Errorf("fillFuncIn0Type %s must eq context.Context", in0Type)
	}

	in1Type := funcType.In(1)
	if in1Type != bizKeysType {
		return fmt.Errorf("fillFuncIn1Type %s must eq bizKeysType %s", in1Type, bizKeysType)
	}
	//检查出参
	if n := funcType.NumOut(); n != 2 {
		return fmt.Errorf("fillFunc num out %d must be 2", n)
	}

	if out0Type := funcType.Out(0); out0Type != outType {
		return fmt.Errorf("fillFuncOut0Type %s must eq outType %s", out0Type, outType)
	}
	out1Type := funcType.Out(1)
	if _, ok := reflect.New(out1Type).Interface().(*error); !ok {
		return fmt.Errorf("fillFuncOut1Kind %s must eq context.Context", out1Type)
	}
	return nil
}

func checkOut(outPtrType, bizKeyType reflect.Type) error {
	if kind := outPtrType.Kind(); kind != reflect.Ptr {
		return fmt.Errorf("outPtr kind %s must be ptr", kind)
	}
	outType := outPtrType.Elem()
	if kind := outType.Kind(); kind != reflect.Map {
		return fmt.Errorf("out kind %s must be map", kind)
	}
	if keyType := outType.Key(); keyType != bizKeyType {
		return fmt.Errorf("out key type %s must eq bizKeyType %s", keyType, bizKeyType)
	}
	return nil
}

/*
入参解释：
1. `bizKeys`必须是`slice`
2. `keyFunc`必须只有一个入参和一个出参，
 * 入参类型必须与`bizKeys`的数组内每个元素的类型相同
 * 出参类型必须是`string`
3. `fillFunc`(不为nil时)必须是2个入参，2个出参
 * 入参顺序：
  1. `ctx context.Context`
  2. `noCachedBizKeys` 未命中的`bizKey`数组，类型必须与`bizKeys`相同
 * 出参顺序：
  1. `noCachedMap map[bizKey类型]bizValue类型`
  2. `err error` 发生错误时，会返回
4. `outPtr` 必须是`map[bizKey类型]bizValue类型`的地址
*/
func (client *Client) BizMGetJsonMapWithFill(ctx context.Context, bizKeys, keyFunc, fillFunc, outPtr interface{}, expiration time.Duration) (err error) {
	defer guard.BeforeCtx(&ctx)(&err)
	entry := logrus.WithContext(ctx).
		WithFields(logrus.Fields{
			"bizKeys":    bizKeys,
			"expiration": expiration,
		})

	//检查bizKeys类型
	bizKeysValue := reflect.ValueOf(bizKeys)
	if kind := bizKeysValue.Kind(); kind != reflect.Slice {
		err = fmt.Errorf("bizKeys kind %s must be slice", kind)
		entry.WithError(err).Error()
		return
	}
	bizKeysType := bizKeysValue.Type()
	bizKeyType := bizKeysType.Elem()

	//检查keyFunc
	keyFuncValue := reflect.ValueOf(keyFunc)
	keyFuncType := reflect.TypeOf(keyFunc)
	if err = checkKeyFunc(keyFuncType, bizKeyType); err != nil {
		entry.WithError(err).Error()
		return
	}

	//检查outPtr
	outPtrType := reflect.TypeOf(outPtr)
	if err = checkOut(outPtrType, bizKeyType); err != nil {
		entry.WithError(err).Error()
		return
	}
	outType := outPtrType.Elem()
	bizValueType := outType.Elem()

	//检查fillFunc
	if fillFunc != nil {
		fillFuncType := reflect.TypeOf(fillFunc)
		if err = checkFillFunc(fillFuncType, bizKeysType, outType); err != nil {
			entry.WithError(err).Error()
			return
		}
	}
	if bizKeysValue.Len() == 0 {
		return //key为空数组就不执行
	}

	var redisKeys []string
	for i := 0; i < bizKeysValue.Len(); i++ {
		out := keyFuncValue.Call([]reflect.Value{bizKeysValue.Index(i)})
		redisKeys = append(redisKeys, out[0].String())
	}
	entry = entry.WithField("redisKeys", redisKeys)

	cmds := client.MGet(redisKeys...)
	entry = entry.WithField("mget-cmds", cmds.String())
	if err = cmds.Err(); err != nil {
		entry.WithError(err).Error()
		return
	}

	noCachedBizKeysValue := reflect.MakeSlice(bizKeysType, 0, 0)

	cachedMapValue := reflect.ValueOf(outPtr).Elem()
	cachedMapValue.Set(reflect.MakeMap(cachedMapValue.Type()))
	for i, val := range cmds.Val() {
		if val == nil {
			noCachedBizKeysValue = reflect.Append(noCachedBizKeysValue, bizKeysValue.Index(i))
		} else {
			newValuePtr := reflect.New(bizValueType).Interface()
			if err = json.Unmarshal([]byte(val.(string)), newValuePtr); err != nil {
				entry.WithError(err).Error()
				return
			}
			cachedMapValue.SetMapIndex(bizKeysValue.Index(i), reflect.ValueOf(newValuePtr).Elem())
		}
	}

	if noCachedBizKeysValue.Len() > 0 && fillFunc != nil {
		fillFuncValue := reflect.ValueOf(fillFunc)
		in := []reflect.Value{reflect.ValueOf(ctx), noCachedBizKeysValue}
		out := fillFuncValue.Call(in)
		if !out[1].IsNil() {
			err = out[1].Interface().(error)
			return
		}
		noCachedMapValue := out[0]
		if noCachedMapValue.Len() > 0 {
			pipe := client.Pipeline()
			defer pipe.Close()

			for iter := noCachedMapValue.MapRange(); iter.Next(); {
				cachedMapValue.SetMapIndex(iter.Key(), iter.Value())
				bs, err := json.Marshal(iter.Value().Interface())
				if err != nil {
					entry.WithError(err).Error()
					return err
				}
				redisKey := keyFuncValue.Call([]reflect.Value{iter.Key()})[0].String()
				pipe.Set(redisKey, string(bs), expiration)
			}
			cmds, err := pipe.Exec()

			var cmdss []string
			for _, cmd := range cmds {
				cmdss = append(cmdss, cmd.String())
				entry = entry.WithField("pipe-set-cmds", cmdss)
			}
			entry.Debug()
			if err != nil {
				entry.WithError(err).Error()
				return err
			}
		}
	}
	return nil
}

func (client *Client) MGetJsonMap(ctx context.Context, keys []string, mapPtr interface{}) (err error) {
	defer guard.BeforeCtx(&ctx)(&err)
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

	entry := logrus.WithContext(ctx).WithField("keys", keys)
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
func (client *Client) MultiSetJson(ctx context.Context, mp map[string]interface{}, expiration time.Duration) (err error) {
	defer guard.BeforeCtx(&ctx)(&err)
	entry := logrus.WithContext(ctx).WithField("mp", mp)

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
