package client_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Oncelane/laneEtcd/src/client"
	"github.com/Oncelane/laneEtcd/src/kvraft"
	"github.com/Oncelane/laneEtcd/src/pkg/laneConfig"
	"github.com/Oncelane/laneEtcd/src/pkg/laneLog"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var ck *client.Clerk

func init() {
	conf := laneConfig.Clerk{}
	laneConfig.Init("config.yml", &conf)
	// laneLog.Logger.Debugln("check conf", conf)
	ck = client.MakeClerk(conf)
	ck.DeleteWithPrefix("")
}

func getMultiClient(num int) []*client.Clerk {
	conf := laneConfig.Clerk{}
	laneConfig.Init("config.yml", &conf)
	clients := make([]*client.Clerk, num)
	for i := range num {
		clients[i] = client.MakeClerk(conf)
	}
	return clients
}

func BenchmarkMultiClient_Get_Put_1(b *testing.B) {
	ck.DeleteWithPrefix("")
	testingNum := []int{1, 5, 25, 100, 1000}
	for num := range testingNum {
		fmt.Printf("%d clients, Get OP\n", num)
		clients := getMultiClient(num)
		for _, ck := range clients {
			go func() {
				for range b.N {
					_, err := ck.Get("logic")
					if err != nil && err != kvraft.ErrNil {
						b.Error(err)
					}
				}
			}()
		}
	}
}
func BenchmarkClient1(b *testing.B) {
	benchmarkClient(b, 1)
}

func BenchmarkClient5(b *testing.B) {
	benchmarkClient(b, 5)
}

func BenchmarkClient25(b *testing.B) {
	benchmarkClient(b, 25)
}

func BenchmarkClient100(b *testing.B) {
	benchmarkClient(b, 100)
}

func BenchmarkClient1000(b *testing.B) {
	benchmarkClient(b, 1000)
}

func benchmarkClient(b *testing.B, clientNum int) {
	fmt.Printf("%d clients, Get OP\n", clientNum)
	b.StopTimer() // 暂停计时
	clients := getMultiClient(clientNum)
	b.ResetTimer() // 重置计时器
	b.StartTimer()
	var wg sync.WaitGroup
	wg.Add(clientNum)

	for _, ck := range clients {
		go func(ck *client.Clerk) {
			defer wg.Done()
			for i := 0; i < b.N; i++ {
				_, err := ck.Get("logic")
				if err != nil && err != kvraft.ErrNil {
					b.Error(err)
				}
			}
		}(ck)
	}
	wg.Wait()
}

// 测试TTL功能

func TestTimeOut(t *testing.T) {
	key := "comet"
	value := []byte("localhost")
	ttl := time.Millisecond * 300
	ck.Put(key, value, ttl)
	laneLog.Logger.Infof("set key [%s] value [%s] TTL[%v]", key, value, ttl)
	laneLog.Logger.Infoln("time.Sleep for 200ms")
	time.Sleep(time.Millisecond * 200)
	v, err := ck.Get("comet")
	if err == kvraft.ErrNil {
		laneLog.Logger.Infoln("err:", err)
	} else {
		laneLog.Logger.Infof("get value [%s]", v)
	}
	laneLog.Logger.Infoln("time.Sleep for 200ms")
	time.Sleep(time.Millisecond * 200)
	v, err = ck.Get("comet")
	if err == kvraft.ErrNil {
		laneLog.Logger.Infoln("err:", err)
	} else {
		laneLog.Logger.Infof("get value [%s]", v)
	}
	laneLog.Logger.Infof("reset key [%s] value [%s] TTL[%v]", key, value, ttl)
	ck.Put("comet", value, time.Millisecond*300)
	v, err = ck.Get("comet")
	if err == kvraft.ErrNil {
		laneLog.Logger.Infoln("err:", err)
	} else {
		laneLog.Logger.Infof("get value [%s]", v)
	}
}

// 单客户端写负载压测数据
func BenchmarkLaneEtcdPut(b *testing.B) {
	key := "logic"
	value := []byte("test")
	for range b.N {
		err := ck.Put(key, value, 0)
		if err != nil {
			b.Error(err)
		}
	}
}

func TestLaneEtcdPut(t *testing.T) {
	key := "logic"
	value := []byte("test")
	for range 4 {
		start := time.Now()
		err := ck.Put(key, value, 0)
		laneLog.Logger.Warnln("client finish put key[%s] spand time:", "logic", time.Since(start))
		if err != nil {
			t.Error(err)
		}
	}
}

// 单客户端读压测数据
func BenchmarkLaneEtcdGet(b *testing.B) {
	for range b.N {
		_, err := ck.Get("logic")
		if err != nil && err != kvraft.ErrNil {
			b.Error(err)
		}
	}
}

func TestLaneEtcdGet(t *testing.T) {
	for range 4 {
		start := time.Now()
		_, err := ck.Get("logic")
		laneLog.Logger.Warnln("client finish put key[%s] spand time:", "logic", time.Since(start))
		if err != nil && err != kvraft.ErrNil {
			t.Error(err)
		}
	}
}

func BenchmarkLaneEtcdGetWithPrefix(b *testing.B) {
	for range b.N {
		_, err := ck.GetWithPrefix("logic")
		if err != nil && err != kvraft.ErrNil {
			b.Error(err)
		}
	}

}

func BenchmarkLaneEtcdDelete(b *testing.B) {
	for range b.N {
		err := ck.Delete("logic")
		if err != nil {
			b.Error(err)
		}
	}

}

var etcd = NewEtcd()

func NewEtcd() *clientv3.Client {
	c, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"127.0.0.1:2379", "127.0.0.1:22379", "127.0.0.1:32379"},
		DialTimeout: time.Second * 5,
	})
	if err != nil {
		laneLog.Logger.Fatalln(err)
	}
	laneLog.Logger.Infoln("success connect etcd")
	return c
}

func BenchmarkEtcdPut(b *testing.B) {

	for range b.N {
		_, err := etcd.Put(context.Background(), "logic", "test")
		if err != nil {
			b.Error(err)
		}
	}
}
func BenchmarkEtcdGet(b *testing.B) {
	for range b.N {
		_, err := etcd.Get(context.Background(), "logic")
		if err != nil {
			b.Error(err)
		}
	}
}
func BenchmarkEtcdGetWithPrefix(b *testing.B) {

	for range b.N {
		_, err := etcd.Get(context.Background(), "logic", clientv3.WithPrefix())
		if err != nil {
			b.Error(err)
		}
	}
}
func BenchmarkEtcdDelete(b *testing.B) {

	for range b.N {
		_, err := etcd.Delete(context.Background(), "logic")
		if err != nil {
			b.Error(err)
		}
	}
}

func TestEtcdPut(t *testing.T) {

	for range 4 {
		start := time.Now()
		_, err := etcd.Put(context.Background(), "logic", "test")
		laneLog.Logger.Warnln("client finish put key[%s] spand time:", "logic", time.Since(start))
		if err != nil {
			t.Error(err)
		}
	}
}
