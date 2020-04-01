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
router.POST("info/:u",bind.Wrap(Info))
router.Run(":11151")
```
## 原理
遍历req对象的成员，当发现存在例如Body成员则解析Body参数，Info函数接口等价于如下OriginalInfo写法：
```go
func OriginalInfo(c *gin.Context) {
	var req struct {
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
	if err := c.ShouldBindJSON(&req.Body); err != nil {
		code.Send(c, nil, code.ClientErrBody.WithError(err))
		//code.Send看起来等价于：
		//c.JSON(http.StatusOK, gin.H{
		//	"ret": 4004,
		//	"msg": "verify body params failed",
		//})
		return
	}
	if err := c.ShouldBindUri(&req.Uri); err != nil {
		code.Send(c, nil, code.ClientErrUri.WithError(err))
		return
	}
	if err := c.ShouldBindQuery(&req.Query); err != nil {
		code.Send(c, nil, code.ClientErrQuery.WithError(err))
		return
	}
	if err := bind.ShouldBindHeader(c.Request.Header, &req.Header); err != nil {
		code.Send(c, nil, code.ClientErrHeader.WithError(err))
		return
	}

	var resp struct {
		R int `json:"r"`
	}

	var err error
	resp.R, err = OriginalInfoBiz(c.Request.Context(), req.Body.B, req.Uri.U, req.Query.Q, req.Header.H)
	code.Send(c, resp, err)
	//此处的code.Send看起来等价于
	//if err != nil {
	//	c.JSON(http.StatusOK, gin.H{
	//		"ret": 5000,
	//		"msg": "server error",
	//	})
	//} else {
	//	c.JSON(http.StatusOK, gin.H{
	//		"ret":  0,
	//		"msg":  "success",
	//		"data": resp,
	//	})
	//}
}

func OriginalInfoBiz(ctx context.Context, b, u, q, h int) (r int, err error) {
	r = b + u + q + h
	return
}
```

## 自由！随意定义响应！
无论你的响应结构是用：
```json
{
  "ret":0,
  "msg":"success",
  "data":"业务数据"
}
```
还是用：
```json
{
  "code":0,
  "result":"ok",
  "xxx":"业务数据"
}
```
甚至是：
```json
{
  "xxx":???,
  "yyy":???
}
```
都是你的自由；  
无论你的http状态码是用`200`、`400`、`500`区分，还是全部用`200`然后用业务`code`区分，都是你的自由!  
### 示例：  
定义入参出参
```go
func UseMySender(ctx context.Context, req struct {
	Query struct {
		Q int `form:"q" binding:"required"`
	}
	Body struct {
		B int `json:"b" binding:"required"`
	}
}) (code int, obj interface{}) {
	switch req.Body.B {
	case 1:
		return http.StatusInternalServerError, gin.H{
			"ret": 5,
			"msg": "server error when use sender",
		}
	case 2:
		return http.StatusOK, gin.H{
			"ret":  0,
			"msg":  "success when use sender",
			"data": 123,
			"tile": 456, //随意定义
		}
	default:
		return http.StatusOK, 1
	}
}
```
定义你自己的入参绑定错误处理Sender、结果处理Sender，然后跑起来
```go
router := gin.New()
router.POST("sender", WithSender(
	func(c *gin.Context, reqFieldName string, bindErr error) {
		c.JSON(http.StatusBadRequest, gin.H{
			"field":  reqFieldName,
			"result": bindErr.Error(),
		})
	},
	func(c *gin.Context, code int, obj interface{}) {
		c.JSON(code, obj)
	}).Wrap(UseMySender))
router.Run(":11152")
```
