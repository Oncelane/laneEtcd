package kvraft

import (
	"context"
	"crypto/rand"
	"laneEtcd/proto/pb"
	"laneEtcd/src/pkg/laneConfig"
	"laneEtcd/src/pkg/laneLog"
	"math/big"
	"sync"
	"time"
)

type Clerk struct {
	servers []*KVClient
	// You will have to modify this struct.
	nextSendLocalId int
	LatestOffset    int32
	clientId        int64
	cTos            []int
	sToc            []int
	conf            laneConfig.Clerk
	mu              sync.Mutex
}

func nrand() int64 {
	max := big.NewInt(int64(1) << 62)
	bigx, _ := rand.Int(rand.Reader, max)
	x := bigx.Int64()
	return x
}

func (c *Clerk) watchEtcd() {

	for {
		for i, kvclient := range c.servers {
			if !kvclient.Valid {
				if kvclient.Realconn != nil {
					kvclient.Realconn.Close()
				}
				k := NewKvClient(c.conf.EtcdAddrs[i])
				if k != nil {
					c.servers[i] = k
					laneLog.Logger.Warnf("update etcd server[%d] addr[%s]", i, c.conf.EtcdAddrs[i])
				}
			}
		}
		time.Sleep(time.Millisecond * 500)
	}
}

func MakeClerk(conf laneConfig.Clerk) *Clerk {
	ck := new(Clerk)
	ck.conf = conf
	// You'll have to add code here
	ck.servers = make([]*KVClient, len(conf.EtcdAddrs))
	for i := range ck.servers {
		ck.servers[i] = new(KVClient)
		ck.servers[i].Valid = false
	}
	ck.nextSendLocalId = int(nrand() % int64(len(conf.EtcdAddrs)))
	ck.LatestOffset = 1
	ck.clientId = nrand()
	ck.cTos = make([]int, len(conf.EtcdAddrs))
	ck.sToc = make([]int, len(conf.EtcdAddrs))
	for i := range ck.cTos {
		ck.cTos[i] = -1
		ck.sToc[i] = -1
	}
	go ck.watchEtcd()
	return ck
}

// fetch the current value for a key.
// returns "" if the key does not exist.
// keeps trying forever in the face of all other errors.
//
// you can send an RPC with code like this:
// ok := ck.servers[i].Call("KVServer.Get", &args, &reply)
//
// the types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. and reply must be passed as a pointer.

func (ck *Clerk) Get(key string) string {
	// You will have to modify this function.
	ck.mu.Lock()
	defer ck.mu.Unlock()
	args := pb.GetArgs{
		Key:          key,
		ClientId:     ck.clientId,
		LatestOffset: ck.LatestOffset,
	}

	count := 0
	lastSendLocalId := -1

	for {

		if ck.nextSendLocalId == lastSendLocalId {
			count++
			if count > 3 {
				count = 0
				ck.changeNextSendId()
			}
		}

		// laneLog.Logger.Infof("clinet [%d] [Get]:send[%d] args[%v]", ck.clientId, ck.nextSendLocalId, args)
		var validCount = 0
		for !ck.servers[ck.nextSendLocalId].Valid {
			ck.changeNextSendId()
			validCount++
			if validCount == len(ck.servers) {
				break
			}
		}
		if validCount == len(ck.servers) {
			laneLog.Logger.Infoln("not exist valid etcd server")
			time.Sleep(time.Second)
			continue
		}
		reply, err := ck.servers[ck.nextSendLocalId].conn.Get(context.Background(), &args)

		//根据reply初始化一下本地server表

		lastSendLocalId = ck.nextSendLocalId
		if err != nil {
			// laneLog.Logger.Infof("clinet [%d] [Get]:[lost] args[%v]", ck.clientId, args)
			//对面失联，那就换下一个继续发
			ck.changeNextSendId()
			continue
		}

		ck.sToc[reply.ServerId] = ck.nextSendLocalId

		switch reply.Err {
		case OK:
			ck.LatestOffset++
			// laneLog.Logger.Infof("clinet [%d] [Get]:[OK] get args[%v] reply[%v]", ck.clientId, args, reply)
			return reply.Value
		case ErrNoKey:
			// laneLog.Logger.Infof("clinet [%d] [Get]:[ErrNo key] get args[%v]", ck.clientId, args)
			return ""
		case ErrWrongLeader:
			// laneLog.Logger.Infof("clinet [%d] [Get]:[ErrWrong LeaderId][%d] get args[%v] reply[%v]", ck.clientId, ck.nextSendLocalId, args, reply)
			//对方也不知道leader
			if reply.LeaderId == -1 {
				//寻找下一个
				ck.changeNextSendId()
			} else {
				//记录对方返回的不可靠leaderId
				if ck.sToc[reply.LeaderId] == -1 { //但是本地还没初始化呢，那就往下一个发
					ck.changeNextSendId()
				} else { //本地还真知道，那下一个就发它所指定的localServerAddress
					ck.nextSendLocalId = ck.sToc[reply.LeaderId]
				}

			}
		case ErrWaitForRecover:
			// laneLog.Logger.Infof("client [%d] [Get]:[Wait for leader recover]", ck.clientId)
			time.Sleep(time.Millisecond * 200)
		default:
			laneLog.Logger.Fatalf("Client [%d] Get reply unknown err [%s](probaly not init)", ck.clientId, reply.Err)
		}

	}
}

// shared by Put and Append.
//
// you can send an RPC with code like this:
// ok := ck.servers[i].Call("KVServer."+op, &args, &reply)
//
// the types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. and reply must be passed as a pointer.
func (ck *Clerk) PutAppend(key string, value string, op string) {
	// You will have to modify this function.
	ck.mu.Lock()
	defer ck.mu.Unlock()
	args := pb.PutAppendArgs{
		Key:          key,
		Value:        value,
		Op:           op,
		ClientId:     ck.clientId,
		LatestOffset: ck.LatestOffset,
	}
	count := 0
	lastSendLocalId := -1
	for {

		if ck.nextSendLocalId == lastSendLocalId {
			count++
			if count > 5 {
				count = 0
				ck.changeNextSendId()
			}
		}

		var validCount = 0
		for !ck.servers[ck.nextSendLocalId].Valid {
			ck.changeNextSendId()
			validCount++
			if validCount == len(ck.servers) {
				break
			}
		}
		if validCount == len(ck.servers) {
			laneLog.Logger.Infoln("not exist valid etcd server")
			time.Sleep(time.Second)
			continue
		}

		laneLog.Logger.Infof("clinet [%d] [PutAppend]:send[%d] args[%v]", ck.clientId, ck.nextSendLocalId, args.String())
		reply, err := ck.servers[ck.nextSendLocalId].conn.PutAppend(context.Background(), &args)
		laneLog.Logger.Debugln("receive etcd:", reply.String(), err)
		//根据reply初始化一下本地server表

		lastSendLocalId = ck.nextSendLocalId
		if err != nil {
			// laneLog.Logger.Infof("clinet [%d] [PutAppend]:[lost] args[%v]", ck.clientId, args)
			//对面失联，那就换下一个继续发
			ck.changeNextSendId()
			continue
		}

		ck.sToc[reply.ServerId] = ck.nextSendLocalId

		switch reply.Err {
		case OK:
			ck.LatestOffset++
			// laneLog.Logger.Infof("clinet [%d] [PutAppend]:[OK] args[%v] reply[%v]", ck.clientId, args, reply)
			return
		case ErrNoKey:
			// laneLog.Logger.Fatalf("Client [%d] [PutAppend]:reply ErrNokey, but should not happend to putAndAppend args", ck.clientId)
		case ErrWrongLeader:
			// laneLog.Logger.Infof("clinet [%d] [PutAppend]:[ErrWrong LeaderId][%d] get args[%v] reply[%v]", ck.clientId, ck.nextSendLocalId, args, reply)
			//对方也不知道leader
			if reply.LeaderId == -1 {
				//寻找下一个
				ck.changeNextSendId()
			} else {
				//记录对方返回的不可靠leaderId
				if ck.sToc[reply.LeaderId] == -1 { //但是本地还没初始化呢，那就往下一个发
					ck.changeNextSendId()
				} else { //本地还真知道，那下一个就发它所指定的localServerAddress
					ck.nextSendLocalId = ck.sToc[reply.LeaderId]
				}

			}
		default:
			laneLog.Logger.Fatalf("Client [%d] [PutAppend]:reply unknown err [%s](probaly not init)", ck.clientId, reply.Err)
		}

	}
}

func (ck *Clerk) changeNextSendId() {
	ck.nextSendLocalId = (ck.nextSendLocalId + 1) % len(ck.servers)
}

func (ck *Clerk) Put(key string, value string) {
	ck.PutAppend(key, value, "Put")
}
func (ck *Clerk) Append(key string, value string) {
	ck.PutAppend(key, value, "Append")
}
