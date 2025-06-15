package client_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/Oncelane/laneEtcd/src/client"
	"github.com/Oncelane/laneEtcd/src/kvraft"
	"github.com/Oncelane/laneEtcd/src/pkg/laneConfig"
	"github.com/Oncelane/laneEtcd/src/pkg/laneLog"
)

var ck *client.Clerk

func init() {
	conf := laneConfig.Clerk{}
	laneConfig.Init("config.yml", &conf)
	// laneLog.Logger.Debugln("check conf", conf)
	ck = client.MakeClerk(conf)
	ck.DeleteWithPrefix("")
}

// Get测试
func TestLaneEtcdGet(t *testing.T) {
	for range 4 {
		start := time.Now()
		_, err := ck.Get("logic")
		laneLog.Logger.Infof("client finish Get key[%s] spand time: %v", "logic", time.Since(start))
		if err != nil && err != kvraft.ErrNil {
			t.Error(err)
		}
	}
}

// Put测试
func TestLaneEtcdPut(t *testing.T) {
	key := "logic"
	value := []byte("test")
	for range 4 {
		start := time.Now()
		err := ck.Put(key, value, 0)
		laneLog.Logger.Infof("client finish put key[%s] spand time: %v", "logic", time.Since(start))
		if err != nil {
			t.Error(err)
		}
	}
}

// TTL功能测试
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

// CAS测试
func TestCAS(t *testing.T) {
	key := "comet"
	ck.Delete(key)
	laneLog.Logger.Infof("CAS set key[%s] ori[%v] dest[%s] ", key, nil, "A")
	ok, err := ck.CAS(key, nil, []byte("A"), 0)
	if err != nil {
		laneLog.Logger.Infoln(err)
	}
	if ok {
		laneLog.Logger.Infof("success")
	} else {
		laneLog.Logger.Infof("fail")
	}

	rt, err := ck.Get(key)
	if err != nil {
		laneLog.Logger.Infoln(err)
	}
	laneLog.Logger.Infof("get key[%s] value[%s]", key, rt)

	laneLog.Logger.Infof("CAS set again key[%s] ori[%v] dest[%s]", key, nil, "B")
	ok, err = ck.CAS(key, nil, []byte("B"), 0)
	if err != nil {
		laneLog.Logger.Infoln(err)
	}
	if ok {
		laneLog.Logger.Infof("success")
	} else {
		laneLog.Logger.Infof("fail")
	}

	rt, err = ck.Get(key)
	if err != nil {
		laneLog.Logger.Infoln(err)
	}
	laneLog.Logger.Infof("get key[%s] value[%s]", key, rt)
	laneLog.Logger.Infof("CAS set again key[%s] ori[%s] dest[%s]", key, "A", "B")
	ok, err = ck.CAS(key, []byte("A"), []byte("B"), 0)
	if err != nil {
		laneLog.Logger.Infoln(err)
	}
	if ok {
		laneLog.Logger.Infof("success")
	} else {
		laneLog.Logger.Infof("fail")
	}

	rt, err = ck.Get(key)
	if err != nil {
		laneLog.Logger.Infoln(err)
	}
	laneLog.Logger.Infof("get key[%s] value[%s]", key, rt)

	err = ck.Delete(key)
	if err != nil {
		laneLog.Logger.Infoln(err)
	}
}

// pipeline测试
func TestBatchApi(t *testing.T) {
	start := time.Now()
	size := 50
	pipe := ck.Pipeline()
	pipe.DeleteWithPrefix("")
	for i := range size {
		pipe.Put("key"+strconv.Itoa(i), []byte(strconv.Itoa(i)), 0)
	}
	err := pipe.Exec()
	if err != nil {
		laneLog.Logger.Fatalln(err)
	}
	laneLog.Logger.Infoln("batch put 50 spand time:", time.Since(start))
	get, err := ck.Get("key1")
	if err != nil {
		laneLog.Logger.Fatalln(err)
	}
	laneLog.Logger.Infoln("get key1: ", get, " spand time:", time.Since(start))

	rets, err := ck.GetWithPrefix("key")
	if err != nil {
		laneLog.Logger.Fatalln(err)
	}
	if len(rets) != size {
		laneLog.Logger.Errorln("batch 50 key not correct real size", len(rets))
	}
	laneLog.Logger.Infoln("get prefixkey: ", rets, " spand time:", time.Since(start))

}
