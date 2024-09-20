package raft

type Op struct {
	ClientId int64 //客户端标识，用于应对重复请求
	Offset   int32 //客户端的请求序列号
	OpType   int   //请求/操作类型
	Key      string
	Value    string
}

const (
	GetT = iota
	PutT
	AppendT
	DelT
	EmptyT //indicate a empty log only use to update leader commitIndex
)
