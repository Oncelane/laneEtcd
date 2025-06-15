package client_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Oncelane/laneEtcd/src/client"
	"github.com/Oncelane/laneEtcd/src/kvraft"
	"github.com/Oncelane/laneEtcd/src/pkg/laneConfig"
	"github.com/Oncelane/laneEtcd/src/pkg/laneLog"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var cks []*client.Clerk

// var opNums []int
var num = 10

func init() {

}

func getMultiClient(num int) []*client.Clerk {
	conf := laneConfig.Clerk{}
	laneConfig.Init("config.yml", &conf)
	clients := make([]*client.Clerk, num)
	// opNums := make([]int, num)
	for i := range num {
		clients[i] = client.MakeClerk(conf)
	}
	return clients
}

func BenchmarkMultiClient(b *testing.B) {
	// fmt.Printf("%d clients, Get OP\n", num)
	var wg sync.WaitGroup
	wg.Add(num)

	for _, ck := range cks {
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

var optotal = 200

// ======吞吐量测试-写负载-不同客户端并发量======
func TestMultiClient_Write1_10(t *testing.T) {
	for _, num := range []int{1, 5, 10, 15, 20, 25, 30} {
		testMultiClient_Write(t, num, []byte("test"), 200)
	}
}

// ======吞吐量测试-写负载-不同数据块大小======
func TestMultiClient_Write1_200(t *testing.T) {
	// data 初始大小为64B，接下来每一次循环，就让data数据块大小变为四倍
	data := []byte("test")
	for range 7 {
		testMultiClient_Write(t, 5, data, 100)
		data = append(data, data...)
		data = append(data, data...)
	}
}

// 辅助函数
func testMultiClient_Write(t *testing.T, num int, data []byte, optotal int) {
	var wg sync.WaitGroup
	cks = getMultiClient(num)
	time.Sleep(time.Second * 2)
	cks[0].DeleteWithPrefix("")
	start := time.Now()
	wg.Add(num)
	for _, ck := range cks {
		go func(ck *client.Clerk) {
			defer wg.Done()
			for range optotal / num {
				err := ck.Put("fix", data, -1)
				if err != nil && err != kvraft.ErrNil {
					t.Error(err)
				}
			}
		}(ck)
	}
	wg.Wait()

	spandTime := time.Since(start)
	t.Logf("%d client finish %d Ops,each data size %d B, spand %v ms, %.5f op/s\n", num, optotal, len(data), spandTime.Milliseconds(), float64(optotal)/float64(spandTime.Seconds()))
}

// ======吞吐量测试-读负载-不同数据块大小======
func TestMultiClient_Read_1_30(t *testing.T) {
	for _, num := range []int{1, 5, 10, 15, 20, 25, 30} {
		testMultiClient_Read(t, num)
	}
}

// 辅助函数
func testMultiClient_Read(t *testing.T, num int) {
	var wg sync.WaitGroup
	cks = getMultiClient(num)
	time.Sleep(time.Second * 2)
	start := time.Now()
	wg.Add(num)
	for _, ck := range cks {
		go func(ck *client.Clerk) {
			defer wg.Done()
			for range optotal / num {
				_, err := ck.Get("test")
				if err != nil && err != kvraft.ErrNil {
					t.Error(err)
				}
			}
		}(ck)
	}
	wg.Wait()

	spandTime := time.Since(start)
	t.Logf("%d client finish %d Ops, spand %v ms, %.5f op/s\n", num, optotal, spandTime.Milliseconds(), float64(optotal)/float64(spandTime.Seconds()))
}

var etcds []*clientv3.Client

func getMultiEtcdClient(num int) []*clientv3.Client {
	clients := make([]*clientv3.Client, num)
	// opNums := make([]int, num)
	for i := range num {
		clients[i] = NewEtcd()
	}
	return clients
}

func NewEtcd() *clientv3.Client {
	c, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"127.0.0.1:2379", "127.0.0.1:22379", "127.0.0.1:32379"},
		DialTimeout: time.Second * 5,
	})
	if err != nil {
		laneLog.Logger.Fatalln(err)
	}
	// laneLog.Logger.Infoln("success connect etcd")
	return c
}

// ======Etcd项目-吞吐量测试-读负载-不同客户端并发量======
func TestMultiClient_Etcd_Write1(t *testing.T) {
	testMultiClient_Etcd_Put(t, 1)
}
func TestMultiClient_Etcd_Write5(t *testing.T) {
	testMultiClient_Etcd_Put(t, 5)
}
func TestMultiClient_Etcd_Write10(t *testing.T) {
	testMultiClient_Etcd_Put(t, 10)
}
func TestMultiClient_Etcd_Write100(t *testing.T) {
	testMultiClient_Etcd_Put(t, 100)
}
func TestMultiClient_Etcd_Write200(t *testing.T) {
	testMultiClient_Etcd_Put(t, 200)
}
func TestMultiClient_Etcd_Write500(t *testing.T) {
	testMultiClient_Etcd_Put(t, 500)
}

func testMultiClient_Etcd_Put(t *testing.T, num int) {
	var wg sync.WaitGroup
	ctx := context.Background()
	etcds = getMultiEtcdClient(num)
	time.Sleep(time.Second * 2)
	start := time.Now()
	wg.Add(num)
	for _, etcd := range etcds {
		go func(etcd *clientv3.Client) {
			defer wg.Done()
			for range optotal / num {
				_, err := etcd.Put(ctx, "fix", "test")
				if err != nil && err != kvraft.ErrNil {
					t.Error(err)
				}
			}
		}(etcd)
	}
	wg.Wait()

	spandTime := time.Since(start)
	t.Logf("%d client finish %d Ops, spand %v ms, %.5f op/s\n", num, optotal, spandTime.Milliseconds(), float64(optotal)/float64(spandTime.Seconds()))
}

// ======Etcd项目-吞吐量测试-写负载-不同客户端并发量======
func TestMultiClient_Etcd_Read1(t *testing.T) {
	testMultiClient_Etcd_Put(t, 1)
}
func TestMultiClient_Etcd_Read5(t *testing.T) {
	testMultiClient_Etcd_Put(t, 5)
}
func TestMultiClient_Etcd_Read10(t *testing.T) {
	testMultiClient_Etcd_Put(t, 10)
}
func TestMultiClient_Etcd_Read100(t *testing.T) {
	testMultiClient_Etcd_Put(t, 100)
}
func TestMultiClient_Etcd_Read200(t *testing.T) {
	testMultiClient_Etcd_Put(t, 200)
}
func TestMultiClient_Etcd_Read500(t *testing.T) {
	testMultiClient_Etcd_Put(t, 500)
}

func testMultiClient_Etcd_Get(t *testing.T, num int) {
	var wg sync.WaitGroup
	ctx := context.Background()
	etcds = getMultiEtcdClient(num)
	time.Sleep(time.Second * 2)
	start := time.Now()
	wg.Add(num)
	for _, etcd := range etcds {
		go func(etcd *clientv3.Client) {
			defer wg.Done()
			for range optotal / num {
				_, err := etcd.Get(ctx, "test")
				if err != nil && err != kvraft.ErrNil {
					t.Error(err)
				}
			}
		}(etcd)
	}
	wg.Wait()

	spandTime := time.Since(start)
	t.Logf("%d client finish %d Ops, spand %v ms, %.5f op/s\n", num, optotal, spandTime.Milliseconds(), float64(optotal)/float64(spandTime.Seconds()))
}
