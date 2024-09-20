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
- batch api

性能：

```sh
#压测命令
go test -benchmem -run=^$ -bench ^Benchmark github.com/Oncelane/laneEtcd/src/cmd/client -benchtime=10s
```

| 压测项目      | laneEtcd    | Etcd        | 耗时与 Etcd 相比 |
| ------------- | ----------- | ----------- | ---------------- |
| Get           | 0.799 ms/op | 0.624 ms/op | +28.0%           |
| GetWithPrefix | 0.762 ms/op | 0.625 ms/op | +21.9%           |
| Put           | 2.071 ms/op | 2.325 ms/op | -10.9%           |
| Delete        | 2.065 ms/op | 2.213 ms/op | -6.7%            |

读写性能均与 Etcd 较为持平，写性能略胜一筹

# 运行服务端

## 编译

```sh
git clone https://github.com/Oncelane/laneEtcd.git
make build
```

## 查看启动配置文件

```yml
# 根目录下的config中自带三实例集群的配置文件 config.yml
addr: 127.0.0.1
port: :51240 # unique
rafts:
  me: 0 # unique
  clients:
    - addr: 127.0.0.1
      port: :32300
    - addr: 127.0.0.1
      port: :32301
    - addr: 127.0.0.1
      port: :32302
databasepath: "data0" # unique
maxraftstate: 100000 # snapshoot时机，当log数目达到此数目时进行快照，释放内存
```

## 运行服务端

```sh
make run
```

## 客户端使用请参考 src/cmd/client/client_test.go

## 防呆 proto 指令

protoc --go_out=.. --go-grpc_out=.. --go-grpc_opt=require_unimplemented_servers=false -I. -Iproto proto/pb/pb.proto
