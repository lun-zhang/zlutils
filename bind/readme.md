# 用bind.Wrap后你就不用再写接口层了！
我认为**接口就要像函数**一样，只要申明了入参和出参结构就行了！  
当我们wiki这样定义一个接口（假设叫POST /info/:u）的请求参数的时候：  
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