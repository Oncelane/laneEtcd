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
- 心跳保活机制，及时发现失联服务
- CPU占用率优化，将算法中的一些轮询机制改为通知机制

性能：

```sh
#压测命令
go test -benchmem -run=^$ -bench ^Benchmark github.com/Oncelane/laneEtcd/src/cmd/client -benchtime=10s
```

| 压测项目      | laneEtcd   | Etcd       | 耗时与 Etcd 相比 |
| ------------- | ---------- | ---------- | ---------------- |
| Get           | 0.76 ms/op | 0.61 ms/op | +24.5%           |
| GetWithPrefix | 0.79 ms/op | 0.61 ms/op | +29.5%           |
| Put           | 1.64 ms/op | 2.28 ms/op | -28.1%           |
| Delete        | 1.68 ms/op | 2.20 ms/op | -23.6%           |

读写性能均与 Etcd 较为持平，写性能略胜一筹

> 此压测性能仅作当前阶段参考，并不意味着本项目性能真的超越 Etcd
>
> etcd 为了支持事务，保存的数据有版本号，可以指定版本读取历史数据（如果不压缩清除历史版本的数据），如果不指定，默认读取最大版本的数据
>
> 而本项目未实现键值对的历史版本，未能很好控制变量
>
> 且本项目的 CPU 占用率高于 Etcd 至少两倍以上，是比较大的缺陷

# 运行服务端

## 编译

```sh
git clone https://github.com/Oncelane/laneEtcd.git
cd laneEtcd
make build
```

## 查看启动配置文件

```yml
# 根目录下的config中自带三个成员的配置文件 config.yml,表示每个实例的启动设置，标识为unique的条目不能在集群成员配置中重复
addr: 127.0.0.1
port: :51240 # unique 客户端与服务端通讯的grpc端口号
rafts:
  me: 0 # unique 有多少个
  endpoints: # 服务端之间沟通使用的grpc端口号
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
