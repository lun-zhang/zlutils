# 日志
## 用trace_id将同一个线程的日志串起来
[从ctx获取trace_id](/xray/)  
logrus提供了WithContext方法传递ctx，因此只需实现logrus提供的Format接口，即可在打印日志前设置trace_id
```go
// import "zlutils/xray"
func (f MyFormatter) Format(e *logrus.Entry) (serialized []byte, err error) {
	traceId := xray.GetTraceId(e.Context)
   	if traceId != "" {
   		e.Data["trace_id"] = traceId
  	}
	...
}
```