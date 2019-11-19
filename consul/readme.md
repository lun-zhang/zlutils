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

## 建议
变量的声明尽量推迟到第一次用到它的地方、减少变量的暴露（这样做的缺点是不运行是不知道缺少哪些配置的）