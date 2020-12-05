# consul配置中心

## 初始化
```go
consul.Init(":8500", "test/service/zlutils")
```
## 读取配置到变量中
一般是降级用的默认配置
```go
var dbConn *gorm.DB
var my mysql.Config
consul.GetJson("mysql",&my)
dbConn = mysql.New(my)
```
### 如果不关心值，只关心执行
一般是用来初始化的配置:  
通常读取配置是用于数据库初始化、日志初始化等，而不是关心配置是什么，那么传入只有一个入参（任意类型）的函数：
```go
var dbConn *gorm.DB
consul.GetJson("mysql", func(my mysql.Config){
	dbConn = mysql.New(my)
})
```
### 配置校验
使用`consul.ValiStruct()`校验结构体，`consul.ValiVar("url")`校验非结构体，校验失败会panic，校验用的是`github.com/go-playground/validator`包  
示例：
```go
consul.ValiStruct().GetJson("mysql", func(my mysql.Config){
	dbConn = mysql.New(my)
})
```
## 监控配置变化
例如新服务上线流量小，想要看看是否会有bug，因此先用debug日志级别，观察一会没问题就修改配置为info或warn级别
```go
var log logger.Config
consul.WatchJson("log_watch",&log,func(){
	logger.Init(log)
})
```
### 不关心值，只关心值变化后执行的函数
通常我们只是想在日志配置变化时，更新一下日志界别，而不关系日志配置的值是什么：
```go
consul.WatchJsonVarious("log_watch",func(log logger.Config){
	logger.Init(log)
})
```
### 配置校验
与GetJson的参数校验类似
```go
consul.ValiStruct().WatchJsonVarious("log_watch",func(log logger.Config){
	logger.Init(log)
})
```
## 配置错误定位
如果key找不到、value解析失败、参数校验失败等，都会有详细的错误日志

## 配置复用
目前每个项目都有自己的配置，好处是不会影响到其他项目，但坏处就是毫无复用可言，
目前最频繁最需要复用的是rpc接口配置，例如`operations_rpc`，以及vcoin加金币等接口  
那么用WithPrefix创建一个新的consul对象，指定新的前缀，
例如我的公共配置放在`test/service/zl_com`目录，
而我的项目目录在`test/service/example`，
那么
```go
consul.Init(":8500","test/service/example")//初始化consul
consul.GetJson("log",func(log logger.config){//这个读的key是 test/service/example/log
	//初始化日志配置
})
consul.WithPrefix("test/service/zl_com").
	GetJson("operations_rpc",func(operationsRpc request.Config){//这个读的key是 test/service/zl_com/operations_rpc
		//注册
})
```
要注意的是公共配置慎用watch

## watch安全
### 如果配置出错, 则不会生效
旧版本使用`WatchJsonVarious("i", &i)`时, 如果配置出错(unmarshal失败或validator失败), 则`i`会被`mysql.SetZero`置为`零值`,   
这一版本已修复, 会先在临时变量进行修改, 成功后才会进行设置
>如果是用的`func`,旧版也没问题, 配置出错时不会调用`func` 
>```go
>WatchJsonVarious("i", func(i *int) {
>  fmt.Println(*i) 
>})
>```
### `WithLocker`并发控制
```go
var m map[int]int
WatchJsonVarious("m", &m)
```
如果修改配置的同时, 其他协程也修改`m`, 则会`panic`, 因为`map`不允许并发修改, 因此需要用锁制造临界区:
```go
mu := &sync.Mutex{} //自行生成一个locker
WithLocker(mu).WatchJsonVarious("m", &m) //watch协程会先Lock()然后才会修改`m`

go func() {
  for {
    mu.Lock()//自己访问`m`时也需Lock()
    m[1] = 1
    mu.Unlock()
  }
}()
```
`watch`的临界区非常小, 只是`unmarshal`和`validate`, 因此无需担心占用锁太久

## 建议
变量的声明尽量推迟到第一次用到它的地方、减少变量的暴露（这样做的缺点是不运行是不知道缺少哪些配置的）