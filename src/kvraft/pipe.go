package kvraft

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"time"

	"github.com/Oncelane/laneEtcd/proto/pb"
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
	// laneLog.Logger.Debugln("batch write:", b.Bytes())
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
func (p *Pipe) Put(key, value string, TTL time.Duration) error {
	d := ValueToData(value, TTL)
	op := raft.Op{
		Key:    key,
		Value:  string(d),
		OpType: int32(pb.OpType_PutT),
	}
	return p.append(op)
}

// func (p *Pipe) Append(key, value string, TTL time.Duration) error {
// 	d := ValueToData(value, TTL)
// 	op := raft.Op{
// 		Key:    key,
// 		Value:  string(d),
// 		OpType: int32(pb.OpType_AppendT),
// 	}
// 	return p.append(op)
// }

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
