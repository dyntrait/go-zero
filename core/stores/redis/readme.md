.
├── conf.go // redis 节点/集群的配置文件，集群的配置文件 Address 是个逗号分割的字符串。其他参数跟单节点的一致
├── conf_test.go
├── hook.go //hook 类似中间件，在 hook 里定义了链路跟踪和 prome 的信息采集逻辑
├── hook_test.go
├── metrics.go //prome 变量的定义
├── metrics_test.go
├── redisblockingnode.go //阻塞 redis 连接，只是在建立连接时把 ReadTimeout option 设置成了 7s
├── redisblockingnode_test.go
├── redisclientmanager.go //用来保存单节点客户端，全局变量，执行命令时，根据 Address 来 map 里查
├── redisclustermanager.go//用来保存集群客户端，全局变量，执行命令时，根据 Address 来 map 里查
├── redisclustermanager_test.go
├── redis.go //redis 结构用来保存 redis 节点的 address、breaker 等信息，不维护 redis 连接，redis 连接在 manager 里管理
├── redislock.go //分布式锁
├── redislock_test.go
├── redistest
│ └── redistest.go
├── redis_test.go
├── scriptcache.go //用来保存 lua 脚本的 hash 值 key 是脚本字符串 value 是 hash 值
└── scriptcache_test.go
