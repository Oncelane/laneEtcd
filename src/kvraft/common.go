package kvraft

import "errors"

// const Debug = false

// func laneLog.Logger.Infof(format string, a ...interface{}) {
// 	if Debug {
// 		laneLog.Logger.Infof(format, a...)
// 	}
// 	return
// }

const (
	ErrOK = iota
	ErrNoKey
	ErrCasFaildInt
	ErrWrongLeader
	ErrWaitForRecover
)

var (
	ErrCASFaild    error = errors.New("etcd: cas op faild")
	ErrNil         error = errors.New("etcd: nil key")
	ErrFaild       error = errors.New("etcd: faild to connect")
	ErrLockFaild   error = errors.New("etcd: lock faild")
	ErrUnLockFaild error = errors.New("etcd: unlock faild")
)

type Err string

// Put or Append
type PutAppendArgs struct {
	Key   string
	Value string
	Op    string // "Put" or "Append"
	// You'll have to add definitions here.
	// Field names must start with capital letters,
	// otherwise RPC will break.
	ClientId     int64
	LatestOffset int32
}

type PutAppendReply struct {
	Err      Err
	LeaderId int
	ServerId int
}

type GetArgs struct {
	Key string
	// You'll have to add definitions here.
	ClientId     int64
	LatestOffset int32
}

type GetReply struct {
	Err      Err
	LeaderId int
	Value    string
	ServerId int
}

type ValueType struct {
	Origin   string
	Value    string
	DeadTime int64
}
