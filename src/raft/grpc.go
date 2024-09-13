package raft

import (
	"laneEtcd/proto/pb"
	"laneEtcd/src/pkg/laneConfig"
	"laneEtcd/src/pkg/laneLog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type RaftEnd struct {
	conf laneConfig.RaftEnd
	conn pb.RaftClient
}

func NewRaftClient(conf laneConfig.RaftEnd) *RaftEnd {
	conn, err := grpc.NewClient(conf.Addr+conf.Port, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		laneLog.Logger.Infoln("Dail faild ", err.Error())
		return nil
	}
	client := pb.NewRaftClient(conn)
	ret := &RaftEnd{
		conn: client,
	}
	return ret
}
