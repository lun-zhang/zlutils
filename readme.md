# 快速开发工具（弱依赖）
这个包含的工具基本覆盖了web服务器开发各个部分，可以方便用户快速开发，
使用户的精力集中于业务而不再为代码细节分心

每个包实现的功能都较为单一，包之间依赖较少，
减少用户的负担，避免用户想使用某一个包的时候发现还得引入另一个包，
断开包之间的依赖也带来了代码冗余的问题，
有些基本包的依赖难以断开，不过这些基本包暴露了一些可供用户自定义的部分，
尽量通用化

写该工具包的原因是遇到如下问题：

# 遇到的问题

## 接口层
主要问题：接口层写来写去都一样，重复劳作好没意思！
1. 接口之间好多共同参数，app类接口需要User-Id等Header参数，admin类接口需要username等Query参数
2. 每个接口都要定义请求Query、Body、Header、Uri参数，又要定义响应参数，太多结构了！
3. 不同接口可能有相同的参数，如何优雅共享结构？
4. 不同接口可能有相同的权限限制，例如app接口必须传用户信息，vb还得用户绑定！
5. 接口层解析的参数又要传给业务层，参数声明又写一遍！

## 数据层

### mysql
1. 每条sql语句执行耗时是怎样？
2. 怎样优雅地让xray跟踪每条sql执行情况
3. 有些拓传数据我想存成json，但每次写入前调用json.Marshal，
取出后调用json.Unmarshal好麻烦啊！这种重复劳作如何解决？
4. 读写分离，aws rds提供了数据库主从库，但得自己代码里区分哪些用主库，哪些用从库
5. 数据库存储时间类型是date，与客户端打交道的时间类型是timestamp(int)，转来转去真麻烦

### redis
能直接存取json结构就好了，这样我就不用关心json解析了

### rpc
1. 写来写去又是一样！
2. 这个接口不是GET又不是POST要我怎么调？
3. 有的接口返回的是ret/msg，有的是code/result，怎么写个统一的rpc函数？

## 错误码
1. 这个函数有服务器错误，又有客户端错误怎么办
2. 客户端错误得返回400，服务端错误得返回500，服务端错误太多得报警
3. 错误码怎么知道冲突没有？
4. 如何灵活增加我自己的错误码？

## debug
如何快速排查问题，有些问题不好重现

### 日志
日志这么多，咋看哪些日志属于同一次请求？  
TODO 用log id把同一个线程的日志串起来

### xray

### 接口层调不通是啥问题？
1. 返回400 verify body params failed，是我body参数哪里错了？服务器同学查下日志吧！
2. 接口调不通，是客户端参数逻辑问题？还是服务器问题？用了docker不好用tcpdump！
3. 测试同学你重现一下bug吧？不好重现怎么办？

## 其他
1. 怎样优雅使用xray监控每个函数的耗时、函数是否发生错误
2. 每个函数都用defer recover保护，将panic转成err，怎样写最简单
5. 我想在consul上配个缓存时间为5分钟，得先转成time.Duration类型的300000000000纳秒，不能像nginx一样配个"5m"吗？

# 如何解决
## 接口层
### 用bind.Wrap后你就不用再写接口层了！
我认为接口就要像函数一样，只要申明了入参和出参结构就行了！  
当我们wiki这样定义一个接口（假设叫info）的请求参的时候：  
body参数：  

| 字段 | 类型 | 必传 |
|-----|------|-----|
|  b  | int  | 是  |

query参数：

| 字段 | 类型 | 必传 |
|-----|------|-----|
|  q  | int  | 是  |

uri参数：

| 字段 | 类型 | 必传 |
|-----|------|-----|
|  u  | int  | 是  |

header参数：

| 字段 | 类型 | 必传 |
|-----|------|-----|
|  H  | int  | 是  |

那么就在代码中定义入参：
```go
struct {
	Body struct {
		B int `json:"b" binding:"required"`
	}
	Uri struct {
		U int `uri:"u" binding:"required"`
	}
	Query struct {
		Q int `form:"q" binding:"required"`
	}
	Header struct {
		H int `header:"h" binding:"required"`
	}
}
```

wiki上定义响应字段：

| 字段 | 类型 | 必返 |
|-----|------|-----|
|  r  | int  | 是  |

定义出参
```go
struct {
	R int `json:"r"`
}
```
组装在一起，完成info接口，功能很简单，就是把入参求和：
```go
func Info(ctx context.Context, req struct {
	Body struct {
		B int `json:"b" binding:"required"`
	}
	Uri struct {
		U int `uri:"u" binding:"required"`
	}
	Query struct {
		Q int `form:"q" binding:"required"`
	}
	Header struct {
		H int `header:"h" binding:"required"`
	}
}) (resp struct {
	R int `json:"r"`
}, err error) {
	resp.R = req.Body.B + req.Uri.U + req.Query.Q + req.Header.H
	return
}
```
最后用bind.Wrap将info接口变成gin.HandlerFunc
```go
router := gin.New()
router.POST("info",bind.Wrap(Info))
router.Run(":11151")
```

### gin.Group控制一组接口的权限、解析共同类型的参数

## 数据层
### mysql
1. 实现gorm.Print接口，加入sql耗时、慢查监控
2. github.com/lun-zhang/gorm 框架
    1. 把非事务的读用从库，非事务的写和事务操作用主库，
    2. 另外新增WithContext(context.Context)函数，从而非破坏性地加入xray跟踪
3. mysql.type包 实现database/sql/driver的Value和Scan方法，
在写入数据库时候将结构转成json格式的[]byte，取出来再解析回去
4. time.Time包 实现了时间类型转成json的时候会转成timestamp(int)，从json解析的时候，会把timestamp(int)转成time.Time
5. time.Duration包 实现json.UnmarshalJSON接口，从json格式的[]byte解出时候，把"5m"解析成5分钟

### redis
封装解析json的函数

### rpc
我认为用http调用其他接口时候，应当像调用自己的函数一样简单，所以实现了request包  
既然要像函数一样，也就是有入参和出参，不管是GET POST，它们入参也就是Body Query Header Uri（这里未实现Uri参数），
出参是响应的Body