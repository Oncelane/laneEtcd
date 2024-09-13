package kvraft

import (
	"laneEtcd/proto/pb"
	"laneEtcd/src/pkg/laneConfig"
	"laneEtcd/src/pkg/laneLog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type KVEnd struct {
	conn pb.KvserverClient
}

func NewKvEnd(conf laneConfig.Kvserver) *KVEnd {
	conn, err := grpc.NewClient(conf.Addr+conf.Port, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		laneLog.Logger.Infoln("Dail faild ", err.Error())
		return nil
	}
	client := pb.NewKvserverClient(conn)

	ret := &KVEnd{
		conn: client,
	}
	return ret
}
