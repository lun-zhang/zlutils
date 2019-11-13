# 调用其他接口时候，应当像调用自己的函数一样简单
既然要像函数一样，也就是有入参和出参，不管是GET POST，它们入参也就是Body Query Header Uri（这里未实现Uri参数），
出参是响应的Body  
例如现在我要调用[bind包下readme中定义的info接口](bind/)，
这个接口太乱了，虽说是POST方法，但是Query Header里为啥要有参数！用ctxhttp.Post没法调，
只能用ctxhttp.Do，并且设置http.Request才行，那么用requst包就很简单了：  
首先确定这个接口的方法、url、超时时间：
```go
//import zt "zlutils/time"
config := request.Config{
	Method: http.MethodPost,
	Url:    "http://localhost:11151/info/4",
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
最后完成调用，响应结果会通过反射解析到resp里，这里ctx支持xray：
```go
if err := req.Do(ctx, &resp); err != nil {
	return
}
```
## 添加query真麻烦怎么办？query只能接收string，所以还得把各种类型转成string?
假设有个批量获取一类图书信息的GET接口，query参数需要书的type和id数组，
于是需要设置一个type，并且循环读取id数组，设置到query里
```go
req := request.Request{
	Config: request.Config{
		Method: http.MethodGet,
		Url:    "http://localhost:11152/book?caller=test",
	},
	Query: request.MSI{
		"type": "history", //一次性设置
	},
}
ids := []int{1, 2}//假设输入的id数组
for _, id := range ids {
	req.AddQuery("id", id) //循环设置
}
```
可以看出这里的id数组是循环设置的，而type参数，我称之为是一次性设置的，直接赋给了map[string]interface{}类型的Query成员，
一次性设置的用况比较大，所以直接当做map来设置就非常方便  
AddQuery接收的value是interface，由to.String转成字符串  
对于Header参数来说，只能是个map，value也由to.String将value转成string
### query参数支持直接传递数组/切片
上面的样例可以这样写
```go
req := request.Request{
	Config: request.Config{
		Method: http.MethodGet,
		Url:    "http://localhost:11152/book?caller=test",
	},
	Query: request.MSI{
		"type": "history", //一次性设置
		"id"  : []int{1, 2},//可以直接输入数组
	},
}
```
### 不用这个包，自己拼接query参数就不是那么方便了
上述query参数拼接完之后应当是
```
http://localhost:11152/book?caller=test&type=history&id=1&id=2
```
由于url本身带了个query参数caller=test，所以得先把url解码解出自带的query参数，然后加入其他参数type、id数组，
最后在编码成最终的url
```go
bookUrl := "http://localhost:11152/book?caller=test"
rawUrl, err := url.Parse(bookUrl)
if err != nil {
	//错误处理
	return
}
query := rawUrl.Query()
ids := []int{1, 2}
query.Add("type", "history")
for _, id := range ids {
	query.Add("id", strconv.Itoa(id))
}
rawUrl.RawQuery = query.Encode()
finalUrl := rawUrl.String()//最终的url
```
