package raft

import (
	"bytes"
	"encoding/gob"
	"unsafe"

	"github.com/Oncelane/laneEtcd/src/common"
	"github.com/Oncelane/laneEtcd/src/pkg/laneLog"
)

type Op struct {
	ClientId int64 //客户端标识，用于应对重复请求
	Offset   int32 //客户端的请求序列号
	OpType   int32 //请求/操作类型
	Key      string
	OriValue []byte
	Entry    common.Entry
	// DeadTIme int64
}

func (o *Op) Size() int {
	return int(unsafe.Sizeof(*o)) + len(o.Key) + int(unsafe.Sizeof(o.Entry)) + len(o.Entry.Value)
}

func (o *Op) Marshal() []byte {
	b := new(bytes.Buffer)
	en := gob.NewEncoder(b)
	err := en.Encode(o)
	if err != nil {
		laneLog.Logger.Fatalln(err)
	}
	return b.Bytes()
}

func (o *Op) Unmarshal(data []byte) {
	b := bytes.NewBuffer(data)
	d := gob.NewDecoder(b)
	err := d.Decode(o)
	if err != nil {
		laneLog.Logger.Fatalf("raft applyArgs.command -> Op 失败,raft_type.Command = %v", data, err)
		return
	}
}
