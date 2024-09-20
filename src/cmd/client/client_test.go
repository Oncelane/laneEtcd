package client_test

import (
	"context"
	"testing"
	"time"

	"github.com/Oncelane/laneEtcd/src/kvraft"
	"github.com/Oncelane/laneEtcd/src/pkg/laneConfig"
	"github.com/Oncelane/laneEtcd/src/pkg/laneLog"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var ck *kvraft.Clerk

func init() {
	conf := laneConfig.Clerk{}
	laneConfig.Init("config.yml", &conf)
	// laneLog.Logger.Debugln("check conf", conf)
	ck = kvraft.MakeClerk(conf)
}

// func TestEtcd(t *testing.T) {

// 	flag.Parse()
// 	conf := laneConfig.Clerk{}
// 	laneConfig.Init(*ConfigPath, &conf)
// 	laneLog.Logger.Debugln("check conf", conf)
// 	ck := kvraft.MakeClerk(conf)
// 	ck.Put("logic", "testLogicAddr")
// 	laneLog.Logger.Infoln("put success")
// 	r, err := ck.Get("logic")
// 	laneLog.Logger.Infoln("get success r= ", r, "err=", err)
// 	r, err = ck.Get("no logic")
// 	laneLog.Logger.Infoln("get success r= ", r, "err=", err)

// 	ck.Put("logic:0", "testLogicAddr0 ")
// 	ck.Put("logic:1", "testLogicAddr1 ")
// 	ck.Put("logic:2", "testLogicAddr2 ")
// 	ck.Put("logic:3", "testLogicAddr3 ")
// 	laneLog.Logger.Infoln("put success")
// 	laneLog.Logger.Infoln("put success")
// 	rt, err := ck.GetWithPrefix("logic")
// 	laneLog.Logger.Infoln("get success r= ", rt, "err=", err)
// }

// func TestMany(t *testing.T) {
// 	start := time.Now()
// 	for i := range 100 {
// 		ck.Put("logic"+strconv.FormatInt(int64(i), 10), "testLogicAddr"+strconv.FormatInt(int64(i), 10))
// 	}
// 	laneLog.Logger.Infoln("put 100 spand time", time.Since(start))
// 	_, err := ck.GetWithPrefix("logic")
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	laneLog.Logger.Infoln("before del:")

// 	for i := range 100 {
// 		ck.Delete("logic" + strconv.FormatInt(int64(i), 10))
// 	}
// 	rt, err := ck.GetWithPrefix("logic")
// 	if err != nil && err != kvraft.ErrNil {
// 		t.Error(err)
// 	}
// 	laneLog.Logger.Infoln("after del", rt)
// }

func BenchmarkLaneEtcdPut(b *testing.B) {

	for range b.N {
		err := ck.Put("logic", "test")
		if err != nil {
			b.Error(err)
		}
	}
}

func TestLaneEtcdPut(t *testing.T) {

	for range 4 {
		start := time.Now()
		err := ck.Put("logic", "test")
		laneLog.Logger.Warnln("client finish put key[%s] spand time:", "logic", time.Since(start))
		if err != nil {
			t.Error(err)
		}
	}
}
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
