package kvraft

import "errors"

// const Debug = false

// func laneLog.Logger.Infof(format string, a ...interface{}) {
// 	if Debug {
// 		laneLog.Logger.Infof(format, a...)
// 	}
// 	return
// }

type Op struct {
	ClientId int64 //客户端标识，用于应对重复请求
	Offset   int32 //客户端的请求序列号
	OpType   int   //请求/操作类型
	Key      string
	Value    string
}

const (
	OK                = "OK"
	ErrNoKey          = "ErrNoKey"
	ErrWrongLeader    = "ErrWrongLeader"
	ErrWaitForRecover = "Wait"
)

var ErrNil error = errors.New("etcd has no key")
var ErrFaild error = errors.New("etcd has faild")

const (
	getT = iota
	putT
	appendT
	emptyT //indicate a empty log only use to update leader commitIndex
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
