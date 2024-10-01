package client_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/Oncelane/laneEtcd/src/pkg/laneLog"
)

func TestBatchApi(t *testing.T) {
	start := time.Now()
	pipe := ck.Pipeline()
	for i := range 1000 {
		pipe.Put("key"+strconv.Itoa(i), strconv.Itoa(i), 0)
	}
	err := pipe.Exec()
	if err != nil {
		laneLog.Logger.Fatalln(err)
	}
	laneLog.Logger.Infoln("batch 1000 spand time:", time.Since(start))
}

func BenchmarkSingleApi(t *testing.B) {
	start := time.Now()
	for i := range t.N {
		ck.Put("key"+strconv.Itoa(i), strconv.Itoa(i), 0)
	}
	laneLog.Logger.Infof("single %f spand time:%v", t.N, time.Since(start))
}
