package client_test

import (
	"testing"

	"github.com/Oncelane/laneEtcd/src/client"
	"github.com/Oncelane/laneEtcd/src/kvraft"
	"github.com/Oncelane/laneEtcd/src/pkg/laneConfig"
)

var ck *client.Clerk

func init() {
	conf := laneConfig.Clerk{}
	laneConfig.Init("config.yml", &conf)
	// laneLog.Logger.Debugln("check conf", conf)
	ck = client.MakeClerk(conf)
	ck.DeleteWithPrefix("")
}

// 写负载压测数据
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

// 读压测数据
func BenchmarkLaneEtcdGet(b *testing.B) {
	for range b.N {
		_, err := ck.Get("logic")
		if err != nil && err != kvraft.ErrNil {
			b.Error(err)
		}
	}
}

// 前缀查询读负载压测
func BenchmarkLaneEtcdGetWithPrefix(b *testing.B) {
	for range b.N {
		_, err := ck.GetWithPrefix("logic")
		if err != nil && err != kvraft.ErrNil {
			b.Error(err)
		}
	}

}

// 删除操作负载压测
func BenchmarkLaneEtcdDelete(b *testing.B) {
	for range b.N {
		err := ck.Delete("logic")
		if err != nil {
			b.Error(err)
		}
	}

}
