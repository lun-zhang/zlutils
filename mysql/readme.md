1. 实现gorm.Print接口，加入sql耗时、慢查监控
2. github.com/lun-zhang/gorm 框架
    1. 把非事务的读用从库，非事务的写和事务操作用主库，
    2. 另外新增WithContext(context.Context)函数，从而非破坏性地加入xray跟踪
3. mysql.type包 实现database/sql/driver的Value和Scan方法，
在写入数据库时候将结构转成json格式的[]byte，取出来再解析回去
4. time.Time包 实现了时间类型转成json的时候会转成timestamp(int)，从json解析的时候，会把timestamp(int)转成time.Time
5. time.Duration包 实现json.UnmarshalJSON接口，从json格式的[]byte解出时候，把"5m"解析成5分钟

# 两行即可增加sql监控
```go
//import "zlutils/mysql"
mysql.InitDefaultMetric(projectName)
//dbConn, err := gorm.Open("mysql", "root:123@/counter?charset=utf8&parseTime=True&loc=Local")
dbConn.LogMode(true).SetLogger(&mysql.Logger{})
```
dbConn无论是由github.com/jinzhu/gorm创建还是github.com/lun-zhang/gorm创建，只要使用了Print(...interface{})接口打印sql，都可用以上两行完加入监控  
效果如下：
```
sum(rate(task_wall_mysql_total[5m])) by (query)
```
![sql耗时](http://hot.onlinemovieweb.com/videobuddy/1569726352-c038a6ab8b55c-sql%E8%80%97%E6%97%B6_w1859_h780.png)
```
sum(rate(task_wall_mysql_latency_millisecond_sum[5m])) by (query) / sum(rate(task_wall_mysql_latency_millisecond_count[5m])) by (query)
```
![sql次数](http://hot.onlinemovieweb.com/videobuddy/1569726356-5150b4db04ce6-sql%E6%AC%A1%E6%95%B0_w1857_h773.png)

WHERE IN(?,...,?)的语句会被getSampleQuery函数替换成WHERE IN(?)，免得参数个数不同变成不同的线（由于getSampleQuery函数写的很简陋，导致VALUES(?,...,?)替换成了VALUES(?)等）  
我实现的Print方法除了sql监控外，还加入了slow sql警告日志：执行耗时超过200ms的sql会打印warning日志  
mysql.MetricCounter和mysql.MetricLatency允许你定义自己喜欢的metric名字以及sql处理方式  
当然你完全也可以通过实现自己的Print方法来加入监控  
