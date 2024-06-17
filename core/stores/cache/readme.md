├── cacheconf.go //缓存的配置文件，支持多节点的
├── cache.go  
├── cachenode.go //单节点缓存类型，实现了 Cache 接口
├── cachenode_test.go
├── cacheopt.go //cache 过期时间，缓存和数据库都查不到时，key 设置为\*。
├── cacheopt_test.go
├── cachestat.go //统计命中、数据库查询次数等
├── cachestat_test.go
├── cache_test.go
├── cleaner.go //删除 key 时，没删成功，弄一个异步任务
├── cleaner_test.go
├── config.go //cache 的配置
├── readme.md
├── util.go
└── util_test.go

此包使用的 redis 连接，\*redis.Redis 作为 cacheNode 的成员。
注意 redis.Redis 表示单节点 redis 客户端或者集群客户端，根据 address 是否多个来区分的。上层应用只管使用，golang 的 redis 库会保证的
cacheCluster 使用一致性缓存容器保存多个 cacheNode.根据 key 查询往哪个 cacheNode 去查询
