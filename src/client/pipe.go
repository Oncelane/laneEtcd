package client

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"time"

	"github.com/Oncelane/laneEtcd/proto/pb"
	"github.com/Oncelane/laneEtcd/src/common"
	"github.com/Oncelane/laneEtcd/src/pkg/laneLog"
	"github.com/Oncelane/laneEtcd/src/raft"
)

type Pipe struct {
	ops  []raft.Op
	size int
	ck   *Clerk
}

func (p *Pipe) Size() int {
	return p.size
}

func (p *Pipe) Marshal() []byte {
	b := new(bytes.Buffer)
	e := gob.NewEncoder(b)
	e.Encode(p.ops)
	return b.Bytes()
}

func (p *Pipe) UnMarshal(data []byte) {
	var ops []raft.Op
	b := bytes.NewBuffer(data)
	d := gob.NewDecoder(b)
	err := d.Decode(&ops)
	if err != nil {
		laneLog.Logger.Fatalln("raw data:", data, err)
	}
	p.ops = ops
}

func (p *Pipe) Delete(key string) error {
	op := raft.Op{
		Key:    key,
		OpType: int32(pb.OpType_DelT),
	}
	return p.append(op)
}

func (p *Pipe) DeleteWithPrefix(prefix string) error {
	op := raft.Op{
		Key:    prefix,
		OpType: int32(pb.OpType_DelWithPrefix),
	}
	return p.append(op)
}
func (p *Pipe) Put(key string, value []byte, TTL time.Duration) error {
	op := raft.Op{
		Key: key,
		Entry: common.Entry{
			Value:    value,
			DeadTime: 0,
		},
		OpType: int32(pb.OpType_PutT),
	}
	if TTL != 0 {
		op.Entry.DeadTime = time.Now().Add(TTL).UnixMilli()
	}
	return p.append(op)
}

func (p *Pipe) Append(key string, value []byte, TTL time.Duration) error {
	op := raft.Op{
		Key: key,
		Entry: common.Entry{
			Value:    value,
			DeadTime: time.Now().Add(TTL).UnixMilli(),
		},
		OpType: int32(pb.OpType_AppendT),
	}
	return p.append(op)
}

func (p *Pipe) append(op raft.Op) error {
	p.ops = append(p.ops, op)
	p.size += op.Size()
	if p.size > pipeLimit {
		return fmt.Errorf("too many pipeline data maxLimit:%d", pipeLimit)
	}
	return nil
}

func (p *Pipe) Exec() error {
	return p.ck.batchWrite(p)
}
