package raft

import "unsafe"

type Op struct {
	ClientId int64 //客户端标识，用于应对重复请求
	Offset   int32 //客户端的请求序列号
	OpType   int32 //请求/操作类型
	Key      string
	Value    string
}

func (o *Op) Size() int {
	return int(unsafe.Sizeof(*o)) + len(o.Key) + len(o.Value)
}
