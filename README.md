# laneETCD

脱离 mit6.5840（前 mit6.840）实现而来，实现 raft 强一致性分布式共识算法，使用 kv 键值对外提供强一致的服务注册等服务

已实现：

- snapshot 持久化，崩溃恢复
- readIndex，读请求不需要记录日志

特性:
- 尝试使用前缀树完成前缀匹配
- del接口
- aof持久化
- 事务


未实现：

- 动态集群
- 租约机制，从节点读

# proto

protoc --go_out=.. --go-grpc_out=.. --go-grpc_opt=require_unimplemented_servers=false -I. -Iproto proto/pb/pb.proto
