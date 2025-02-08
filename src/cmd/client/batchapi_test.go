package client_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/Oncelane/laneEtcd/src/kvraft"
	"github.com/Oncelane/laneEtcd/src/pkg/laneLog"
)

func TestBatchApi(t *testing.T) {
	start := time.Now()
	size := 1000
	pipe := ck.Pipeline()
	pipe.DeleteWithPrefix("")
	for i := range size {
		pipe.Put("key"+strconv.Itoa(i), []byte(strconv.Itoa(i)), 0)
	}
	err := pipe.Exec()
	if err != nil {
		laneLog.Logger.Fatalln(err)
	}
	laneLog.Logger.Infoln("batch 1000 spand time:", time.Since(start))
	get, err := ck.Get("key1")
	if err != nil {
		laneLog.Logger.Fatalln(err)
	}
	laneLog.Logger.Infoln("key1=", get)

	rets, err := ck.GetWithPrefix("key")
	if err != nil {
		laneLog.Logger.Fatalln(err)
	}
	if len(rets) != size {
		laneLog.Logger.Errorln("batch 1000 key not correct real size", len(rets))
	}

}

func TestGetWithPrefix(t *testing.T) {
	ck.DeleteWithPrefix("")
	err := ck.Put("key:1", []byte("1"), 0)
	if err != nil {
		laneLog.Logger.Fatalln(err)
	}
	err = ck.Put("key:2", []byte("2"), 0)
	if err != nil {
		laneLog.Logger.Fatalln(err)
	}
	pipe := ck.Pipeline()
	pipe.Put("key:3", []byte("3"), 0)
	err = pipe.Exec()
	if err != nil {
		laneLog.Logger.Fatalln(err)
	}
	rowdata, err := ck.GetWithPrefix("key")
	if err != nil {
		laneLog.Logger.Fatalln(err)
	}
	if len(rowdata) != 3 {
		laneLog.Logger.Fatalln("wrong count")
	}
	ck.DeleteWithPrefix("key")
	_, err = ck.GetWithPrefix("key")
	if err != kvraft.ErrNil {
		laneLog.Logger.Fatalln(err)
	}

}

func TestKvs(t *testing.T) {
	ck.DeleteWithPrefix("")
	err := ck.Put("key:1", []byte("1"), 0)
	if err != nil {
		laneLog.Logger.Fatalln(err)
	}
	err = ck.Put("key:2", []byte("2"), 0)
	if err != nil {
		laneLog.Logger.Fatalln(err)
	}
	pipe := ck.Pipeline()
	pipe.Put("key:3", []byte("3"), 0)
	err = pipe.Exec()
	if err != nil {
		laneLog.Logger.Fatalln(err)
	}
	rowdata, err := ck.Keys()
	if err != nil {
		laneLog.Logger.Fatalln(err)
	}
	if len(rowdata) != 3 {
		laneLog.Logger.Fatalln("wrong count")
	}
	for i := range rowdata {
		if string(rowdata[i].Key) != "key:"+strconv.Itoa(i+1) {
			laneLog.Logger.Fatalln("wrong key:", string(rowdata[i].Key), "expect:", "key:"+strconv.Itoa(i+1))
		}
	}
	ck.DeleteWithPrefix("key")
	_, err = ck.Keys()
	if err != kvraft.ErrNil {
		laneLog.Logger.Fatalln(err)
	}

}

// func BenchmarkSingleApi(t *testing.B) {
// 	start := time.Now()
// 	for i := range t.N {
// 		ck.Put("key"+strconv.Itoa(i), strconv.Itoa(i), 0)
// 	}
// 	laneLog.Logger.Infof("single %f spand time:%v", t.N, time.Since(start))
// }
