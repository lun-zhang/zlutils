# xray

## xray.GetTraceId从ctx中获取trace_id
跟踪ctxhttp.Do(ctx, xray.Client(client), request)发现发出请求时设置Header里的TraceId取自于seg.DownstreamHeader()：
/data/apps/go/pkg/mod/github.com/aws/aws-xray-sdk-go@v1.0.0-rc.5.0.20180720202646-037b81b2bf76/xray/segment_model.go 134行  
