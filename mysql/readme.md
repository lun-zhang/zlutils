1. 实现gorm.Print接口，加入sql耗时、慢查监控
2. github.com/lun-zhang/gorm 框架
    1. 把非事务的读用从库，非事务的写和事务操作用主库，
    2. 另外新增WithContext(context.Context)函数，从而非破坏性地加入xray跟踪
3. mysql.type包 实现database/sql/driver的Value和Scan方法，
在写入数据库时候将结构转成json格式的[]byte，取出来再解析回去
4. time.Time包 实现了时间类型转成json的时候会转成timestamp(int)，从json解析的时候，会把timestamp(int)转成time.Time
5. time.Duration包 实现json.UnmarshalJSON接口，从json格式的[]byte解出时候，把"5m"解析成5分钟
