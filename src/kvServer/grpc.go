package kvraft

import (
	"laneEtcd/proto/pb"
	"laneEtcd/src/pkg/laneLog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type KVClient struct {
	Valid    bool
	conn     pb.KvserverClient
	Realconn *grpc.ClientConn
}

func NewKvClient(addr string) *KVClient {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		laneLog.Logger.Infoln("Dail faild ", err.Error())
		return nil
	}
	client := pb.NewKvserverClient(conn)

	ret := &KVClient{
		Valid: true,
		conn:  client,
	}
	return ret
}
