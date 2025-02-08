package buntdbx_test

import (
	"bytes"
	"strconv"
	"testing"
	"time"

	"github.com/Oncelane/laneEtcd/src/common"
	buntdbx "github.com/Oncelane/laneEtcd/src/pkg/buntDBX"
)

func TestGetPutEntry(t *testing.T) {
	db := buntdbx.NewDB()

	put := common.Entry{
		Value:    common.StringToBytes("value1"),
		DeadTime: time.Now().Add(time.Second).UnixMilli(),
	}
	err := db.PutEntry("test1", put)
	if err != nil {
		t.Fatal(err)
	}

	ret, err := db.GetEntry("test1")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(put.Value, ret.Value) || put.DeadTime != ret.DeadTime {
		t.Fatal("wrong get and put")
	}
}

func TestGetWithPrefixEntry(t *testing.T) {
	db := buntdbx.NewDB()
	puts := []common.Entry{{
		Value:    common.StringToBytes("1"),
		DeadTime: time.Now().Add(time.Second).UnixMilli(),
	}, {
		Value:    common.StringToBytes("2"),
		DeadTime: time.Now().Add(time.Second).UnixMilli(),
	}, {
		Value:    common.StringToBytes("3"),
		DeadTime: time.Now().Add(time.Second).UnixMilli(),
	},
	}
	for i := range puts {
		err := db.PutEntry("gorpc:"+strconv.Itoa(i), puts[i])
		if err != nil {
			t.Fatal(err)
		}
	}

	rets, err := db.GetEntryWithPrefix("gorpc:")
	if err != nil {
		t.Fatal(err)
	}
	if len(rets) != len(puts) {
		t.Fatalf("wrong getprefiex put.len= %d but ret.len=%d \nput = %v \nret = %v", len(puts), len(rets), puts, rets)
	}
	for i := range puts {
		if !bytes.Equal(puts[i].Value, rets[i].Value) || puts[i].DeadTime != rets[i].DeadTime {
			t.Fatal("wrong getprefix")
		}
	}
}

func TestKeysAndKVs(t *testing.T) {
	db := buntdbx.NewDB()
	puts := []common.Entry{{
		Value:    common.StringToBytes("1"),
		DeadTime: time.Now().Add(time.Second).UnixMilli(),
	}, {
		Value:    common.StringToBytes("2"),
		DeadTime: time.Now().Add(time.Second).UnixMilli(),
	}, {
		Value:    common.StringToBytes("3"),
		DeadTime: time.Now().Add(time.Second).UnixMilli(),
	},
	}
	for i := range puts {
		err := db.PutEntry("gorpc:"+strconv.Itoa(i), puts[i])
		if err != nil {
			t.Fatal(err)
		}
	}
	rets1, err := db.Keys(0, 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(rets1) != len(puts) {
		t.Fatal("wrong length")
	}
	for i := range rets1 {
		if string(rets1[i].Key) != "gorpc:"+strconv.Itoa(i) {
			t.Fatalf("wrong key")
		}
	}
	rets2, err := db.KVs(0, 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(rets2) != len(puts) {
		t.Fatal("wrong length")
	}
	for i := range rets2 {
		if rets2[i].Key != "gorpc:"+strconv.Itoa(i) {
			t.Fatalf("wrong key")
		}
		if !bytes.Equal(rets2[i].Entry.Value, puts[i].Value) {
			t.Fatalf("wrong value ret:%v put:%v", rets2[i].Entry.Value, puts[i].Value)
		}
	}
}
