# laneETCD

独立于 MIT6.5840 的类 Etcd 分布式 kv 数据库，底层实现 raft 强一致性分布式共识算法，提供高可用的服务注册等服务

底层数据库：buntDB，支持 pattern 查询和删除，支持硬盘持久化

- get/delete by prefix：前缀范围查询和删除
- TTL-key：键值对过期时间
- MetaTags：元标签，可用于扩展版本控制，流量染色，负载均衡
- pipeline：使用类似 redis 的 pipeline 语义进行批量操作
- CAS api：使用 CAS 语义进行 Set 操作；未来计划封装成 SetEX，SetNX 等接口；
- Lock api：分布式锁，可以直接调用客户端提供的 api 直接使用分布式锁了，已配合 watchDog 租约机制，默认应用程序意外关闭，断开心跳连接后五秒自动销毁 lock；

特性：

- 集群部署
- snapshot 持久化，崩溃恢复
- readIndex，读请求不需要记录日志
- WatchDog: 心跳保活机制，及时发现失联服务
- CPU 占用率优化，将算法中的一些轮询机制改为通知机制

未来实现：

- 增加脚本功能，可以从网页端上传 lua 脚本，执行自动逻辑
- 非强一致性读：租约机制，从节点读
- duplicateMap 内存占用问题，即 clientId 租期机制

下一阶段目标：

使用新的分布式公式协议`epaxos`重写共识算法层

## 构建

```sh
git clone https://github.com/Oncelane/laneEtcd.git
cd laneEtcd
make build
make run
## 第一次运行会生成配置文件，检查后再次make run 运行
#  make run
```

## 调用 api

提供 go client 和 go gateway 服务两种 api

### 1. go 客户端：

```sh
cd src/cmd/client
ls
```

client_test.go 为例，使用 client.MakeClerk 即可生成客户端

```go
func init() {
	conf := laneConfig.Clerk{}
	laneConfig.Init("config.yml", &conf)
	// laneLog.Logger.Debugln("check conf", conf)
	ck = client.MakeClerk(conf)
}
/*
client api一览
func (ck *client.Clerk) Append(key string, value []byte, TTL time.Duration) error
func (ck *client.Clerk) CAS(key string, origin []byte, dest []byte, TTL time.Duration) (bool, error)
func (ck *client.Clerk) Delete(key string) error
func (ck *client.Clerk) DeleteWithPrefix(prefix string) error
func (ck *client.Clerk) Get(key string) ([]byte, error)
func (ck *client.Clerk) GetWithPrefix(key string) ([][]byte, error)
func (ck *client.Clerk) KVs() ([]common.Pair, error)
func (ck *client.Clerk) KVsWithPage(pageSize int, pageIndex int) ([]common.Pair, error)
func (ck *client.Clerk) Keys() ([]common.Pair, error)
func (ck *client.Clerk) KeysWithPage(pageSize int, pageIndex int) ([]common.Pair, error)
func (ck *client.Clerk) Lock(key string, TTL time.Duration) (id string, err error)
func (ck *client.Clerk) Pipeline() *client.Pipe
func (ck *client.Clerk) Put(key string, value []byte, TTL time.Duration) error
func (ck *client.Clerk) Unlock(key string, id string) (bool, error)
func (ck *client.Clerk) WatchDog(key string, value []byte) (cancel func())
*/
```

### 2. go gateway 提供 http 服务

```go
var gate *gateway.Gateway

func init() {
	var conf laneConfig.Gateway
	laneConfig.Init("config.yml", &conf) // 在config中配置基础网址和ip，port等参数
	gate = gateway.NewGateway(conf)
}
func main() {
	gate.Run()
}
/*
  http 接口一览
	r.GET(g.conf.BaseUrl+"/keys", g.keys)
	r.GET(g.conf.BaseUrl+"/key", g.get)
	r.GET(g.conf.BaseUrl+"/keysWithPrefix", g.getWithPrefix)
	r.GET(g.conf.BaseUrl+"/kvs", g.kvs)
	r.POST(g.conf.BaseUrl+"/put", g.put)
	r.POST(g.conf.BaseUrl+"/putCAS", g.putCAS)
	r.DELETE(g.conf.BaseUrl+"/key", g.del)
	r.DELETE(g.conf.BaseUrl+"/keysWithPrefix", g.delWithPrefix)
*/
```

性能：

```sh
#压测命令
go test -benchmem -run=^$ -bench ^Benchmark github.com/Oncelane/laneEtcd/src/cmd/client -benchtime=10s
```

压测环境一:
CPU: i7-10750H CPU @ 2.60GHz

| 压测项目      | laneEtcd   | Etcd       | 耗时与 Etcd 相比 |
| ------------- | ---------- | ---------- | ---------------- |
| Get           | 0.76 ms/op | 0.61 ms/op | +24.5%           |
| GetWithPrefix | 0.79 ms/op | 0.61 ms/op | +29.5%           |
| Put           | 1.64 ms/op | 2.28 ms/op | -28.1%           |
| Delete        | 1.68 ms/op | 2.20 ms/op | -23.6%           |

压测环境二:
CPU: AMD Ryzen 5 5600H with Radeon Graphics

| 压测项目      | laneEtcd   | Etcd       | 耗时与 Etcd 相比 |
| ------------- | ---------- | ---------- | ---------------- |
| Get           | 0.84 ms/op | 0.81 ms/op | +3.7%            |
| GetWithPrefix | 0.87 ms/op | 0.81 ms/op | +7.4%            |
| Put           | 1.85 ms/op | 5.37 ms/op | -65.5%           |
| Delete        | 1.89 ms/op | 5.28 ms/op | -64.2%           |

从压测数据可以看到 laneEtcd 读写性能均与 Etcd 较为持平，在写性能上略胜一筹

尤其是在较低配置的笔记本上，Etcd 性能下降非常明显，而 laneEtcd 性能只有小小的下降。

> 此压测性能仅作当前阶段参考，并不意味着本项目性能真的超越 Etcd
>
> etcd 3.0 为了支持事务，保存的数据有版本号，可以指定版本读取历史数据（如果不压缩清除历史版本的数据），如果不指定，默认读取最大版本的数据，且因为需要保存的数据量增加，从内存存储改成了磁盘存储，使用了 BoltDB 数据库
>
> 而本项目仍然使用内存存储且只通过 CAS 支持简单的事务。

# 测试截图

laneEtcd

![1](./docs/laneEtcd.png)

Etcd

![2](./docs/Etcd.png)

配置较低的笔记本上的压测源数据：

![3](./docs/最新压测结果之家里笔记本神秘莫测结果.png)

> laneEtcd 的性能变化不大，但是 Etcd 的性能相比在另一台高配一点的笔记本上的数据就有些奇怪，推测是主动限制了 cpu 占用率
> 但是使用 htop 命令的时候，laneEtcd 和 Etcd 两者的 cpu 占用率差距很小，12 核均在 20~30%波动，因此暂时将此数据中 Etcd 的部分搁置，降低参考意义。

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
