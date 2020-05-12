# 快速开发工具（弱依赖）
这个包含的工具基本覆盖了web服务器开发各个部分，可以方便用户快速开发，
使用户的精力集中于业务而不再为代码细节分心

每个包实现的功能都较为单一，包之间依赖较少，
减少用户的负担，避免用户想使用某一个包的时候发现还得引入另一个包，
断开包之间的依赖也带来了代码冗余的问题，
有些基本包的依赖难以断开，不过这些基本包暴露了一些可供用户自定义的部分，
尽量通用化

写该工具包的原因是遇到如下问题：

# 接口层
主要问题：接口层写来写去都一样，重复劳作好没意思！
1. [接口之间好多共同参数，app类接口需要User-Id等Header参数，admin类接口需要username等Query参数](session/)
2. 每个接口都要定义请求Query、Body、Header、Uri参数，又要定义响应参数，太多结构了！
3. 不同接口可能有相同的参数，如何优雅共享结构？
4. 不同接口可能有相同的权限限制，例如app接口必须传用户信息，vb还得用户绑定！
5. 接口层解析的参数又要传给业务层，参数声明又写一遍！
## 解决办法
1. [用bind.Wrap后你就不用再写接口层了！](bind/)
2. gin.Group控制一组接口的权限、解析共同类型的参数

# 数据层
## [mysql](mysql/)
1. 每条sql语句执行耗时是怎样？
2. 怎样优雅地让xray跟踪每条sql执行情况
3. 有些拓传数据我想存成json，但每次写入前调用json.Marshal，
取出后调用json.Unmarshal好麻烦啊！这种重复劳作如何解决？
4. 读写分离，aws rds提供了数据库主从库，但得自己代码里区分哪些用主库，哪些用从库
5. 数据库存储时间类型是date，与客户端打交道的时间类型是timestamp(int)，转来转去真麻烦

## [redis](redis/)
能直接存取json结构就好了，这样我就不用关心json解析了

## [rpc](request/)
1. 写来写去又是一样！
2. 这个接口不是GET又不是POST要我怎么调？
3. 有的接口返回的是ret/msg，有的是code/result，怎么写个统一的rpc函数？

# [错误码](code/)
1. 这个函数有服务器错误，又有客户端错误怎么办
2. 客户端错误得返回400，服务端错误得返回500，服务端错误太多得报警
3. 错误码怎么知道冲突没有？
4. 如何灵活增加我自己的错误码？

# debug
如何快速排查问题，有些问题不好重现

## [日志](logger/)
日志这么多，咋看哪些日志属于同一次请求？  

## [xray](xray/)

## [接口层调不通是啥问题？](code/)
1. 返回400 verify body params failed，是我body参数哪里错了？服务器同学查下日志吧！
2. 接口调不通，是客户端参数逻辑问题？还是服务器问题？用了docker不好用tcpdump！
3. 测试同学你重现一下bug吧？不好重现怎么办？

# 其他
1. [怎样优雅使用xray监控每个函数的耗时、函数是否发生错误](guard/)
2. [每个函数都用defer recover保护，将panic转成err，怎样写最简单](guard/)
5. [我想在consul上配个缓存时间为5分钟，得先转成time.Duration类型的300000000000纳秒，不能像nginx一样配个"5m"吗？](time/)

# 如何使用此工具包
go.mod中用replace
```
require zlutils v0.0.0
replace zlutils v0.0.0 => xlbj-gitlab.xunlei.cn/oversea/zlutils/v7 v7 //go build时会找到v7最新版本
```
代码中这样导入：`import "zlutils/time"`，这样做的好处是：
1. ***避免同时引入本项目的不同版本***，导致类型/变量值不匹配

坏处是：

## 工具包在公司私有gitlab上，go mod拉取报错
该项目在公司私有gitlab，因此安全起见都是用ssh拉的代码，但是go.mod中用的url是https的`xlbj-gitlab.xunlei.cn/oversea/zlutils`，会报错：
```
go: errors parsing go.mod:
/data/project/proxy_spider/go.mod:17: invalid module version xlbj-gitlab.xunlei.cn/oversea/zlutils/v7: git ls-remote -q https://xlbj-gitlab.xunlei.cn/oversea/zlutils.git in /data/project/go/pkg/mod/cache/vcs/6156227698db14dbc4a3f0737ba273596653cb201e8541df7305578248fa4fcd: exit status 128:
        fatal: could not read Username for 'https://xlbj-gitlab.xunlei.cn': terminal prompts disabled
```
因此需要设置一下本地git配置，让git在拉取公司代码时，将`https://xlbj-gitlab.xunlei.cn`替换成`git@xlbj-gitlab.xunlei.cn`:
```
~$ cat .gitconfig
[url "git@xlbj-gitlab.xunlei.cn:"]
    insteadOf = https://xlbj-gitlab.xunlei.cn/
```

## 1.13以上报错：reading sum.golang.org 410 Gone
```
go: xlbj-gitlab.xunlei.cn/oversea/zlutils/v7@v7.15.0/go.mod: verifying module: xlbj-gitlab.xunlei.cn/oversea/zlutils/v7@v7.15.0/go.mod: reading https://sum.golang.org/lookup/xlbj-gitlab.xunlei.cn/oversea/zlutils/v7@v7.15.0: 410 Gone
        server response: not found: xlbj-gitlab.xunlei.cn/oversea/zlutils/v7@v7.15.0: unrecognized import path "xlbj-gitlab.xunlei.cn/oversea/zlutils/v7": https fetch: Get "https://xlbj-gitlab.xunlei.cn/oversea/zlutils/v7?go-get=1": dial tcp 47.103.68.13:443: connect: connection refused
```
1.13之后设置了默认的GOSUMDB=sum.golang.org  
所以要执行以下命令关闭：
```
go env -w GOSUMDB=off
```
