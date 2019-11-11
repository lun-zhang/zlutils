# 错误码

前提约定：任何到达服务器的请求，即使发生任何错误，都返回http.status=200，具体错误用ret区分，
这样做的目的是：
1. 与网关的错误做区分，方便其他人在调用我们接口的时候，知道不是200的请求是不需要服务器同学查日志的
2. app内页面发起请求是通过客户端提供的接口，客户端发现http.status!=200时候，会把错误信息丢掉，
页面端便无法得知错误信息

## 错误码结构
目前我们的服务器最常用的响应结构（例如成功）是
```json
{
  "ret":0,
  "msg":"success",
  "data":业务数据，任意类型
}
```
## 错误码分类
服务器可能返回的错误类型分两大类：服务器错误和客户端错误  
客户端错误又分为2大类： 客户端参数错误和逻辑错误  
目前客户端参数错误有以下几种：
1. body参数错误
2. query参数错误
3. header参数错误
4. uri参数错误

客户端逻辑错则每个服务各不一样，因此提供了自定义错误码的函数code.Add，
这个函数会检查错误码是否冲突

## 如何与现有go错误处理风格兼容？
go推荐的错误处理风格是函数返回的最后一个参数类型为error：
```go
func f() (data interface{}, err error) {
    //处理，有错误就返回
    return
}
```
调用这个函数，如果err!=nil则返回
```go
if _, err = f(); err != nil {
	return
}
```
因此错误码需要实现Error接口
```go
func (code Code) Error() string {
	if code.Err != nil {
		return code.Err.Error()
	}
	return fmt.Sprintf("ret: %d, msg: %s", code.Ret, code.Msg)
}
```
## 响应错误码
虽然现在不需要调用code.Send了(因为[用bind.Wrap后你就不用再写接口层了！](bind/))，但还是说下：  
现在只需要调用code.Send(c,data,err)，就能识别err是服务器错误还是客户端错误，
另外有错误的时候，不需要也不应该把data数据返回  
### 没有错误码时是这样处理的
```go
data, err := f()
if err != nil {
	c.JSON(http.StatusInternalServerError, nil)
} else {
	c.JSON(http.StatusOK, data)
}
```
要区分客户端错误或服务器错误是这样的：
```go
data, errServer, errClient := g()
if errClient != nil {
	c.JSON(http.StatusBadRequest, nil)
} else if errServer != nil {
	c.JSON(http.StatusInternalServerError, nil)
} else {
	c.JSON(http.StatusOK, data)
}
```

## 只能收到verify body params failed或者server error，能否返回更多信息帮助调用者排查问题？
通常在第一次调用他人接口的时候，会因为参数错误而调不通，如果只是返回了verify body params failed这样的错误信息，
还需要自己去对着文档一个一个看字段类型、是否必传等等，或者直接让服务器同学过来看，那服务器同学如果把请求打印日志了，倒也可以看出个问题，
但是如果在测试阶段，把错误信息，例如哪个字段解析失败等信息返给调用者便可以：
1. 简单的类型错误调用者便可以知道，就不必麻烦服务器同学了
2. 复杂错误，有时是客户端逻辑错误，例如活动结束后不该再参与，服务器同学看到（调用者通过钉钉发来的）错误信息便马上知道问题所在，不必再查日志

对于server error的服务器错误，也返回详细信息，方便服务器同学自己定位问题（例如gorm插入不存在的字段，便可知自己忘记扩列了），而不必再查日志  
在调用code.Send的时候，只有识别到目前处于开发/测试阶段 才会返回详细的错误信息，因为有些错误信息非常敏感，可能危及服务器安全，因此正式环境不会返回详细错误信息
### 如何携带详细信息
WithError(error)和WithErrorf(format,args...)函数满足了用户添加详细的错误信息  
例如现在有个活动服务，定义了一个活动不存的的错误码，这是个客户端错误
```go
var clientErrNoActivity = code.Add(4101, "no activity")
```
活动不存在有多种可能，例如客户端传来的活动id是错的，或者活动已经结束了
```go
if 用活动id查不到活动 {
	return clientErrNoActivity.WithErrorf("no activity_id:%d", activityId)
}
//如果查到的活动，但是活动已结束
if 活动已结束 {
	return clientErrNoActivity.WithErrorf("activity_id %d has ended at:%s, now is: %s", activityId, activity.EndAt, now)
}
```

### 输出trace_id方便定位
如果接口发生少量错误，还能通过过滤error关键字找到日志，但是  
如果测试同学说调了一次接口，返回是200，ret=0，但是数据逻辑不正确，如何定位到这次请求？  
那么返回输出trace_id吧([logger包](logger/)已支持trace_id，
[gorm](https://github.com/lun-zhang/gorm/tree/v1.13.3)打印日志时也写入了trace_id，如何使用参考[mysql包](mysql/))： 
```json
{
    "ret": 0,
    "msg": "success",
    "trace_id": "1-5dc3c17c-2ffcc981d21a5cf13698baea",
    "data": 1
}
```
由于trace_id也算敏感信息，因此用code.MidRespWithTraceId中间件来控制指定接口是否输出（同code.MidRespWithErr）

## msg支持多语言
当ret!=0时，客户端将msg作为toast内容弹出，支持多语言：
```go
var codeClientErrTaskLimitTotal = code.Add(4101, code.MLS{
    code.LangEn: "Sorry, today's special are all sold-out. Pls come early tomorrow.",
    code.LangHi: "क्षमा करें, आज का विशेष बोनस सभी बिक चुके हैं। कल जल्दी आना।",
})
```

## TODO
### 将错误码改成接口
以实现不同结构的错误码，例如
```json
{
  "code":0,
  "result":"ok",
  "data":业务数据
}
```
### 将错误码配置在consul
1. msg支持多语言后，如果经常改动，需要放在consul
2. 错误码写在wiki似乎不好维护
