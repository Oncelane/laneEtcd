package client

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/gob"
	"math/big"
	"sync"
	"time"

	"github.com/Oncelane/laneEtcd/proto/pb"
	"github.com/Oncelane/laneEtcd/src/common"
	"github.com/Oncelane/laneEtcd/src/kvraft"
	"github.com/Oncelane/laneEtcd/src/pkg/laneConfig"
	"github.com/Oncelane/laneEtcd/src/pkg/laneLog"
	"github.com/google/uuid"
)

var pipeLimit int = 1024 * 4000

type Clerk struct {
	servers []*kvraft.KVClient
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
				k := kvraft.NewKvClient(c.conf.EtcdAddrs[i])
				if k != nil {
					c.servers[i] = k
					// laneLog.Logger.Warnf("update etcd server[%d] addr[%s]", i, c.conf.EtcdAddrs[i])
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
	ck.servers = make([]*kvraft.KVClient, len(conf.EtcdAddrs))
	for i := range ck.servers {
		ck.servers[i] = new(kvraft.KVClient)
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

func (ck *Clerk) doGetValue(key string, withPrefix bool) ([][]byte, error) {
	args := pb.GetArgs{
		Key:          key,
		ClientId:     ck.clientId,
		LatestOffset: ck.LatestOffset,
		WithPrefix:   withPrefix,
		Op:           pb.OpType_GetT,
	}
	return ck.read(&args)
}

func (ck *Clerk) doGetKV(key string, withPrefix bool, op pb.OpType, pageSize, pageIndex int) ([]common.Pair, error) {
	args := pb.GetArgs{
		Key:          key,
		ClientId:     ck.clientId,
		LatestOffset: ck.LatestOffset,
		WithPrefix:   withPrefix,
		PageSize:     int32(pageSize),
		PageIndex:    int32(pageIndex),
		Op:           op,
	}
	datas, err := ck.read(&args)
	if err != nil {
		return nil, err
	}
	rets := make([]common.Pair, len(datas))
	for i := range rets {
		d := gob.NewDecoder(bytes.NewBuffer(datas[i]))
		d.Decode(&rets[i])
	}
	return rets, nil
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

func (ck *Clerk) read(args *pb.GetArgs) ([][]byte, error) {
	// You will have to modify this function.
	ck.mu.Lock()
	defer ck.mu.Unlock()
	totalCount := 0
	count := 0
	lastSendLocalId := -1
	for {
		totalCount++
		if totalCount > 2*len(ck.servers)*3 {
			return nil, kvraft.ErrFaild
		}
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
		reply, err := ck.servers[ck.nextSendLocalId].Conn.Get(context.Background(), args)

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
		case kvraft.ErrOK:
			ck.LatestOffset++
			// laneLog.Logger.Infof("clinet [%d] [Get]:[OK] get args[%v] reply[%v]", ck.clientId, args, reply)
			if len(reply.Value) == 0 {
				return nil, kvraft.ErrNil
			}

			return reply.Value, nil
		case kvraft.ErrNoKey:
			// laneLog.Logger.Infof("clinet [%d] [Get]:[ErrNo key] get args[%v]", ck.clientId, args)
			ck.LatestOffset++
			return nil, kvraft.ErrNil
		case kvraft.ErrWrongLeader:
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
		case kvraft.ErrWaitForRecover:
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
func (ck *Clerk) write(key string, value, oriValue []byte, TTL time.Duration, op int32) error {
	// You will have to modify this function.

	ck.mu.Lock()
	defer ck.mu.Unlock()
	args := pb.PutAppendArgs{
		Key:          key,
		Value:        value,
		OriValue:     oriValue,
		Op:           op,
		ClientId:     ck.clientId,
		LatestOffset: ck.LatestOffset,
	}
	if TTL == 0 {
		args.DeadTime = 0
	} else {
		args.DeadTime = time.Now().Add(TTL).UnixMilli()
	}
	count := 0
	lastSendLocalId := -1
	totalCount := 0
	for {
		totalCount++
		if totalCount > 2*len(ck.servers)*3 {
			return kvraft.ErrFaild
		}
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
			// laneLog.Logger.Infoln("not exist valid etcd server")
			time.Sleep(time.Millisecond * 10)
			continue
		}

		// laneLog.Logger.Infof("clinet [%d] [PutAppend]:send[%d] args[%v]", ck.clientId, ck.nextSendLocalId, args.String())
		reply, err := ck.servers[ck.nextSendLocalId].Conn.PutAppend(context.Background(), &args)
		// laneLog.Logger.Debugln("receive etcd:", reply.String(), err)
		//根据reply初始化一下本地server表

		lastSendLocalId = ck.nextSendLocalId
		if err != nil {
			// laneLog.Logger.Infof("clinet [%d] [PutAppend]:[lost] args[%v] err:", ck.clientId, args, err)
			//对面失联，那就换下一个继续发
			ck.changeNextSendId()
			continue
		}

		ck.sToc[reply.ServerId] = ck.nextSendLocalId

		switch reply.Err {
		case kvraft.ErrOK:
			ck.LatestOffset++
			// laneLog.Logger.Infof("clinet [%d] [Get]:[OK] get args[%v] reply[%v]", ck.clientId, args, reply)
			return nil
		case kvraft.ErrNoKey:
			// laneLog.Logger.Infof("clinet [%d] [Get]:[ErrNo key] get args[%v]", ck.clientId, args)
			ck.LatestOffset++
			return kvraft.ErrNil
		case kvraft.ErrCasFaildInt:
			ck.LatestOffset++
			return kvraft.ErrCASFaild
		case kvraft.ErrWrongLeader:
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

func (ck *Clerk) Put(key string, value []byte, TTL time.Duration) error {
	return ck.write(key, value, nil, TTL, int32(pb.OpType_PutT))
}

func (ck *Clerk) Append(key string, value []byte, TTL time.Duration) error {
	return ck.write(key, value, nil, TTL, int32(pb.OpType_AppendT))
}

func (ck *Clerk) Delete(key string) error {
	return ck.write(key, nil, nil, 0, int32(pb.OpType_DelT))
}

func (ck *Clerk) DeleteWithPrefix(prefix string) error {
	return ck.write(prefix, nil, nil, 0, int32(pb.OpType_DelWithPrefix))
}

func (ck *Clerk) CAS(key string, origin, dest []byte, TTL time.Duration) (bool, error) {
	err := ck.write(key, dest, origin, TTL, int32(pb.OpType_CAST))
	if err != nil {
		if err == kvraft.ErrCASFaild {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (ck *Clerk) batchWrite(p *Pipe) error {
	return ck.write("", p.Marshal(), nil, 0, int32(pb.OpType_BatchT))
}

func (ck *Clerk) Pipeline() *Pipe {
	return &Pipe{
		ck: ck,
	}
}

func (ck *Clerk) Get(key string) ([]byte, error) {
	r, err := ck.doGetValue(key, false)
	if err != nil {
		return nil, err
	}
	if len(r) == 1 {
		return r[0], nil
	}
	return nil, kvraft.ErrNil
}

func (ck *Clerk) GetWithPrefix(key string) ([][]byte, error) {
	r, err := ck.doGetValue(key, true)
	if err != nil {
		return nil, err
	}
	if len(r) != 0 {
		return r, nil
	}
	return nil, kvraft.ErrNil
}

func (ck *Clerk) Keys() ([]common.Pair, error) {
	return ck.doGetKV("", true, pb.OpType_GetKeys, 0, 0)
}

func (ck *Clerk) KVs() ([]common.Pair, error) {
	return ck.doGetKV("", true, pb.OpType_GetKVs, 0, 0)
}

func (ck *Clerk) KeysWithPage(pageSize, pageIndex int) ([]common.Pair, error) {
	return ck.doGetKV("", true, pb.OpType_GetKeys, pageSize, pageIndex)
}

func (ck *Clerk) KVsWithPage(pageSize, pageIndex int) ([]common.Pair, error) {
	return ck.doGetKV("", true, pb.OpType_GetKVs, pageSize, pageIndex)
}

// TODO 当TTL为零时，启动watchDog机制
func (ck *Clerk) Lock(key string, TTL time.Duration) (id string, err error) {
	r, err := uuid.NewRandom()
	if err != nil {
		laneLog.Logger.Fatalln(err)
		return "", err
	}
	ok, err := ck.CAS(key, nil, []byte(r.String()), TTL)
	if err != nil {
		laneLog.Logger.Fatalln(err)
		return "", err
	}
	if !ok {
		return "", nil
	}

	return r.String(), nil
}

func (ck *Clerk) Unlock(key, id string) (bool, error) {
	ok, err := ck.CAS(key, []byte(id), nil, 0)
	if err != nil {
		laneLog.Logger.Fatalln(err)
		return false, err
	}
	if !ok {
		return false, nil
	}
	return true, nil
}

func (ck *Clerk) WatchDog(key string, value []byte) (cancel func()) {
	// 选择最小5s的生存周期
	var (
		TTL  = time.Second * 5
		flag = new(bool)
	)
	ck.Put(key, value, TTL)
	*flag = false
	go func(key string, ori []byte) { //每500ms续期一次WatchDog
		for {
			curData := make([]byte, len(ori))
			copy(curData, ori)
			if *flag {
				return
			}
			ck.Put(key, curData, TTL)
			time.Sleep(TTL)
		}
	}(key, value)
	return func() {
		*flag = true
	}
}
