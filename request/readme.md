我认为用http调用其他接口时候，应当像调用自己的函数一样简单，所以实现了request包  
既然要像函数一样，也就是有入参和出参，不管是GET POST，它们入参也就是Body Query Header Uri（这里未实现Uri参数），
出参是响应的Body  
例如现在我要调用上面定义的info接口（假设响应是code/result/data格式）
这个接口太乱了，虽说是POST方法，但是Query Header里为啥要有参数！用ctxhttp.Post没法调，
只能用ctxhttp.Do，并且设置http.Request才行，用requst包就很简单了：  
首先确定这个接口的方法、url、超时时间：
```go
//import zt "zlutils/time"
config := request.Config{
	Method: http.MethodPost,
	Url:    "http://localhost:11151/info/:u",
	Client: &request.ClientConfig{
		Timeout: zt.Duration{Duration: time.Second * 2},
	},
}
```
实际上接口配置通常是写到consul上的：
```json
{
  "method": "POST",
  "url": "http://localhost:11151/info/4",
  "client": {
    "timeout": "2s"
  }
}
```

然后传入参数
```go
//import "zlutils/request"
req := request.Request{
	Config: config,
	Query: request.MSI{
		"q": 1,
	},
	Header: request.MSI{
		"H": 2,
	},
	Body: 3,
}
```
实现Check接口，自定义返回错误码为ret/msg，以及错误表达形式  
这里认为ret不为0就是错误的，同时把msg信息拼接成一条错误信息
```go
type RetMsg struct {
	Ret   int `json:"ret"`
	Msg string `json:"msg"`
}

func (m RetMsg) Check() error {
	if m.Ret != 0 {
		return fmt.Errorf("ret:%d msg:%s", m.Ret, m.Msg)
	}
	return nil
}
```
有了响应错误码结构，就能定义响应参数了：
```go
var resp struct {
	RetMsg
	Data struct {
		R int `json:"r"`
	}
}
```
最后完成调用，响应结果会通过反射解析到resp里：
```go
if err := req.Do(ctx, &resp); err != nil {
	return
}
```
