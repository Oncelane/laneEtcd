# laneETCD

脱离 mit6.5840（前 mit6.840）实现而来，实现 raft 强一致性分布式共识算法，使用 kv 键值对外提供强一致的服务注册等服务

已实现：

- 集群部署
- snapshot 持久化，崩溃恢复
- readIndex，读请求不需要记录日志

特性:

- 压缩前缀树存储
- 支持前缀范围查询

暂未实现：

- 动态集群
- 租约机制，从节点读

性能：

```sh
goos: linux
goarch: amd64
pkg: github.com/Oncelane/laneEtcd/src/cmd/client
cpu: Intel(R) Core(TM) i7-10750H CPU @ 2.60GHz
BenchmarkPut-12                      253          13567027 ns/op
BenchmarkGet-12                     3387           1064236 ns/op
BenchmarkGetWithPrefix-12           3439           1081428 ns/op
BenchmarkDelete-12                   210          16171981 ns/op
PASS
```

# proto

protoc --go_out=.. --go-grpc_out=.. --go-grpc_opt=require_unimplemented_servers=false -I. -Iproto proto/pb/pb.proto
