package kvraft

import (
	"bytes"
	"context"
	"encoding/gob"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Oncelane/laneEtcd/proto/pb"
	"github.com/Oncelane/laneEtcd/src/common"
	buntdbx "github.com/Oncelane/laneEtcd/src/pkg/buntDBX"
	"github.com/Oncelane/laneEtcd/src/pkg/laneConfig"
	"github.com/Oncelane/laneEtcd/src/pkg/laneLog"
	"github.com/Oncelane/laneEtcd/src/raft"

	"google.golang.org/grpc"
)

type KVServer struct {
	mu      sync.Mutex
	me      int
	rf      *raft.Raft
	applyCh chan raft.ApplyMsg
	dead    int32 // set by Kill()

	maxraftstate int // snapshot if log grows this big

	persister *raft.Persister
	// Your definitions here.

	//duplicateMap: use to handle mulity RPC request
	// duplicateMap map[int64]duplicateType

	lastAppliedIndex int //最近添加到状态机中的raft层的log的index
	//lastInclude
	lastIncludeIndex int
	//log state machine
	db common.DB

	//缓存的log, seq->index,reply
	duplicateMap map[int64]duplicateType

	grpc *grpc.Server

	lastIndexCh chan int
}

type duplicateType struct {
	Offset int32
	// Reply     string
	CASResult bool
}

func (kv *KVServer) Get(_ context.Context, args *pb.GetArgs) (reply *pb.GetReply, err error) {
	reply = new(pb.GetReply)
	reply.Err = ErrWrongLeader
	reply.LeaderId = int32(kv.rf.GetleaderId())
	reply.ServerId = int32(kv.me)

	//判断自己是不是leader
	if _, ok := kv.rf.GetState(); ok {
		// laneLog.Logger.Infof("server [%d] [info] i am leader", kv.me)
	} else {
		// laneLog.Logger.Infof("server [%d] [info] i am not leader ,leader is [%d]", kv.me, reply.LeaderId)
		return
	}

	//判断自己有没有从重启中恢复完毕状态机
	if !kv.rf.IisBack {
		laneLog.Logger.Infof("server [%d] [recovering] reject a [Get]🔰 args[%v]", kv.me, args)
		reply.Err = ErrWaitForRecover
		b := new(bytes.Buffer)
		e := gob.NewEncoder(b)
		e.Encode(raft.Op{
			OpType: int32(pb.OpType_EmptyT),
		})
		if err != nil {
			laneLog.Logger.Fatalln(err)
		}
		kv.rf.Start(b.Bytes())
		return reply, nil
	}

	readLastIndex := kv.rf.GetCommitIndex()
	term := kv.rf.GetTerm()
	//需要发送一轮心跳获得大多数回复，只是为了确定没有一个任期更加新的leader，保证自己的数据不是脏的
	if kv.rf.CheckIfDepose() {
		reply.Err = ErrWrongLeader
		return
	}

	//return false ,但是进入下面代码段的时候，发现自己又不是leader了，捏麻麻的
	kv.mu.Lock()
	defer kv.mu.Unlock()
	//跟raft层之间的同步问题，raft刚当选leader的时候，还没有

	if kv.lastAppliedIndex >= readLastIndex && kv.rf.GetLeader() && term == kv.rf.GetTerm() {
		var value [][]byte
		switch args.Op {
		case pb.OpType_GetT:
			if args.WithPrefix {
				entrys, err := kv.db.GetEntryWithPrefix(args.Key)
				if err != nil {
					laneLog.Logger.Fatalf("database GetEntryWithPrefix faild:%s", err)
				}

				value = make([][]byte, 0, len(entrys))
				for _, e := range entrys {
					value = append(value, e.Value)
				}
			} else {
				v, err := kv.db.GetEntry(args.Key)
				if err != common.ErrNotFound {
					value = append(value, v.Value)
				}
			}
		case pb.OpType_GetKeys:
			ret, err := kv.db.Keys(int(args.PageSize), int(args.PageIndex))
			if err != nil {
				laneLog.Logger.Fatalln(err)
			}
			value = make([][]byte, 0, len(ret))
			for i := range ret {
				var buf bytes.Buffer
				gob.NewEncoder(&buf).Encode(&ret[i])
				value = append(value, buf.Bytes())
				// buf.Reset()
			}
		case pb.OpType_GetKVs:
			ret, err := kv.db.KVs(int(args.PageSize), int(args.PageIndex))
			if err != nil {
				laneLog.Logger.Fatalln(err)
			}
			value = make([][]byte, 0, len(ret))
			for i := range ret {
				var buf bytes.Buffer
				gob.NewEncoder(&buf).Encode(&ret[i])
				value = append(value, buf.Bytes())
				// buf.Reset()
			}
		}

		if len(value) == 0 {
			reply.Err = ErrNoKey
			return
		}
		reply.Err = ErrOK
		reply.Value = value
		// laneLog.Logger.Infof("server [%d] [Get] [ok] lastAppliedIndex[%d] readLastIndex[%d]", kv.me, kv.lastAppliedIndex, readLastIndex)
		// laneLog.Logger.Infof("server [%d] [Get] [Ok] the get args[%v] reply[%v]", kv.me, args, reply)
	} else {
		reply.Err = ErrWaitForRecover
		// laneLog.Logger.Infof("server [%d] [Get] [ErrWaitForRecover] kv.lastAppliedIndex < readLastIndex args[%v] reply[%v]", kv.me, *args, *reply)
	}

	// laneLog.Logger.Infof("server [%d] [Get] [NoKey] the get args[%v] reply[%v]", kv.me, args, reply)
	// laneLog.Logger.Infof("server [%d] [map] -> %v", kv.me, kv.db)

	return reply, nil
}

func (kv *KVServer) PutAppend(_ context.Context, args *pb.PutAppendArgs) (reply *pb.PutAppendReply, err error) {
	// start := time.Now()
	// laneLog.Logger.Infof("server [%d] [PutAppend] 📨receive a args[%v]", kv.me, args.String())
	// defer func() {
	// 	laneLog.Logger.Infof("server [%d] [PutAppend] 📨complete a args[%v] spand time:%v", kv.me, args.String(), time.Since(start))
	// }()
	reply = new(pb.PutAppendReply)
	// Your code here.
	reply.LeaderId = int32(kv.rf.GetleaderId())
	reply.Err = ErrWrongLeader
	reply.ServerId = int32(kv.me)

	if _, ok := kv.rf.GetState(); ok {
		// laneLog.Logger.Infof("server [%d] [info] i am leader", kv.me)
	} else {
		// laneLog.Logger.Infof("server [%d] [info] i am not leader ,leader is [%d]", kv.me, reply.LeaderId)
		return
	}
	// v := DateToValue(args.Value)

	op := raft.Op{
		ClientId: args.ClientId,
		Offset:   args.LatestOffset,
		OpType:   args.Op,
		Key:      args.Key,
		OriValue: args.OriValue,
		Entry: common.Entry{
			Value:    args.Value,
			DeadTime: args.DeadTime,
		},
	}

	//start前需要查看本地log缓存是否有seq

	//这里通过缓存提交，一方面提高了kvserver应对网络错误的回复速度，另一方面进行了第一层的重复检测
	//但是注意可能同时有两个相同的getDuplicateMap通过这里
	kv.mu.Lock()
	if args.LatestOffset < kv.duplicateMap[args.ClientId].Offset {
		kv.mu.Unlock()
		//laneLog.Logger.Debugln("pass", kv.me)
		return
	}
	if args.LatestOffset == kv.duplicateMap[args.ClientId].Offset {
		if op.OpType != int32(pb.OpType_CAST) {
			reply.Err = ErrOK
		} else {
			if kv.duplicateMap[args.ClientId].CASResult {
				reply.Err = ErrOK
			} else {
				reply.Err = ErrCasFaildInt
			}
		}
		//laneLog.Logger.Debugln("pass", kv.me)
		kv.mu.Unlock()
		return
	}
	kv.mu.Unlock()

	//没有在本地缓存发现过seq
	//向raft提交操作
	// laneLog.Logger.Debugln("raw data:", []byte(op.Value))
	// data, err := json.Marshal(op)

	index, term, isleader := kv.rf.Start(op.Marshal())

	if !isleader {
		return
	}

	kv.rf.SendAppendEntriesToAll()
	// laneLog.Logger.Infof("server [%d] submit to raft key[%v] value[%v]", kv.me, op.Key, op.Value)
	//提交后阻塞等待
	//等待applyCh拿到对应的index，比对seq是否正确
	startWait := time.Now()
	for !kv.killed() {

		kv.mu.Lock()

		if index <= kv.lastAppliedIndex {
			//双重防重复
			if args.LatestOffset < kv.duplicateMap[args.ClientId].Offset {
				//laneLog.Logger.Debugln("pass", kv.me)
				kv.mu.Unlock()
				return
			}
			if args.LatestOffset == kv.duplicateMap[args.ClientId].Offset {
				if op.OpType != int32(pb.OpType_CAST) {
					reply.Err = ErrOK
				} else {
					if kv.duplicateMap[args.ClientId].CASResult {
						reply.Err = ErrOK
					} else {
						reply.Err = ErrCasFaildInt
					}
				}
				//laneLog.Logger.Debugln("pass", kv.me)
				kv.mu.Unlock()
				return
			}

			laneLog.Logger.Infof("server [%d] [PutAppend] appliedIndex available :PutAppend index[%d] lastAppliedIndex[%d]", kv.me, index, kv.lastAppliedIndex)
			if term != kv.rf.GetTerm() {
				//term不匹配了，说明本次提交失效
				kv.mu.Unlock()
				//laneLog.Logger.Debugln("pass", kv.me)
				return
			} //term匹配，说明本次提交一定是有效的

			reply.Err = ErrOK
			laneLog.Logger.Infof("server [%d] [PutAppend] success args.index[%d]", kv.me, index)
			kv.mu.Unlock()
			if _, isleader := kv.rf.GetState(); !isleader {
				reply.Err = ErrWrongLeader
			}
			//laneLog.Logger.Debugln("pass", kv.me)
			return
		}
		kv.mu.Unlock()
		select {
		case <-kv.lastIndexCh:
			// 阻塞等待
		case <-time.After(time.Millisecond * 500):
			laneLog.Logger.Infof("server [%d] [PutAppend] fail [time out] args.index[%d]", kv.me, index)
			return
		}
		// 因为time.After可能会在超时前多次被重置，所以还需要在外层额外做保证
		if time.Since(startWait).Milliseconds() > 500 {
			laneLog.Logger.Infof("server [%d] [PutAppend] fail [time out] args.index[%d]", kv.me, index)
			return
		}
	}
	return reply, nil
}

// state machine
// 将value重新转换为 Op，添加到本地db中
func (kv *KVServer) HandleApplych() {
	for !kv.killed() {
		select {
		case raft_type := <-kv.applyCh:
			//laneLog.Logger.Debugln("pass", kv.me)
			if kv.killed() {
				return
			}
			kv.mu.Lock()
			if raft_type.CommandValid {
				kv.HandleApplychCommand(raft_type)
				kv.checkifNeedSnapshot(raft_type.CommandIndex)
				kv.lastAppliedIndex = raft_type.CommandIndex
				for {
					select {
					case kv.lastIndexCh <- raft_type.CommandIndex:
					default:
						goto APPLYBREAK
					}
				}
			APPLYBREAK:
				// laneLog.Logger.Debugln("pass", kv.me, "  raft_type.CommandIndex=", raft_type.CommandIndex)
			} else if raft_type.SnapshotValid {
				laneLog.Logger.Infof("📷 server [%d] receive raftSnapshotIndex[%d]", kv.me, raft_type.SnapshotIndex)
				kv.HandleApplychSnapshot(raft_type)
			} else {
				laneLog.Logger.Fatalf("Unrecordnized applyArgs type")
			}
			kv.mu.Unlock()
		}

	}
}

func (kv *KVServer) HandleApplychCommand(raft_type raft.ApplyMsg) {
	OP := new(raft.Op)
	OP.Unmarshal(raft_type.Command)
	if OP.OpType == int32(pb.OpType_EmptyT) {
		return
	}
	if OP.Offset <= kv.duplicateMap[OP.ClientId].Offset {
		laneLog.Logger.Infof("⛔server [%d] [%v] lastapplied[%v]find in the cache and discard %v", kv.me, OP, kv.lastAppliedIndex, kv.db)
		return
	}
	kv.duplicateMap[OP.ClientId] = duplicateType{
		Offset: OP.Offset,
	}
	switch OP.OpType {
	case int32(pb.OpType_PutT):
		//更新状态机
		//有可能有多个start重复执行，所以这一步要检验重复

		err := kv.db.PutEntry(OP.Key, OP.Entry)
		if err != nil {
			laneLog.Logger.Fatalf("database putEntry faild:%s", err)
		}

		// laneLog.Logger.Infof("server [%d] [Update] [Put]->[%s,%s] [map] -> %v", kv.me, op_type.Key, op_type.Value, kv.db)
		// laneLog.Logger.Infof("server [%d] [Update] [Put]->[%s : %s] ", kv.me, op_type.Key, op_type.Value)
	case int32(pb.OpType_AppendT):

		ori, _ := kv.db.GetEntry(OP.Key)
		var buffer bytes.Buffer

		// 写入数据
		buffer.Write(ori.Value)
		buffer.Write(OP.Entry.Value)

		// 获取拼接结果
		result := buffer.Bytes()
		err := kv.db.PutEntry(OP.Key, common.Entry{
			Value:    result,
			DeadTime: OP.Entry.DeadTime,
		})
		if err != nil {
			laneLog.Logger.Fatalf("database putEntry faild:%s", err)
		}
		// laneLog.Logger.Infof("server [%d] [Update] [Append]->[%s : %s]", kv.me, op_type.Key, op_type.Value)

	case int32(pb.OpType_DelT):
		kv.db.Del(OP.Key)

	case int32(pb.OpType_DelWithPrefix):
		kv.db.DelWithPrefix(OP.Key)

	case int32(pb.OpType_CAST):
		ori, _ := kv.db.GetEntry(OP.Key)
		if bytes.Equal(ori.Value, OP.OriValue) {
			if len(OP.Entry.Value) == 0 {
				kv.db.Del(OP.Key)
			} else {
				kv.db.PutEntry(OP.Key, OP.Entry)
			}
			kv.duplicateMap[OP.ClientId] = duplicateType{
				Offset:    OP.Offset,
				CASResult: true,
			}
		}
	case int32(pb.OpType_GetT):

		laneLog.Logger.Fatalf("日志中不应该出现getType")

	case int32(pb.OpType_BatchT):
		var ops []raft.Op
		b := bytes.NewBuffer([]byte(OP.Entry.Value))
		// laneLog.Logger.Debugln("receive batch data:", b.Bytes())
		d := gob.NewDecoder(b)
		err := d.Decode(&ops)
		if err != nil {
			laneLog.Logger.Fatalln("raw data:", []byte(OP.Entry.Value), err)
		}
		for _, op := range ops {
			switch op.OpType {
			case int32(pb.OpType_PutT):
				kv.db.PutEntry(op.Key, op.Entry)
			case int32(pb.OpType_AppendT):
				ori, _ := kv.db.GetEntry(op.Key)

				var buffer bytes.Buffer

				// 写入数据
				buffer.Write(ori.Value)
				buffer.Write(op.Entry.Value)

				// 获取拼接结果
				result := buffer.Bytes()
				kv.db.PutEntry(op.Key, common.Entry{
					Value:    result,
					DeadTime: op.Entry.DeadTime,
				})
			case int32(pb.OpType_DelT):
				kv.db.Del(op.Key)
			case int32(pb.OpType_DelWithPrefix):
				kv.db.DelWithPrefix(op.Key)
			}
			// laneLog.Logger.Infof("exec batch op: %+v", op)
		}

	default:
		laneLog.Logger.Fatalf("日志中出现未知optype = [%d]", OP.OpType)
	}

}

// 被动快照,follower接受从leader传来的snapshot
func (kv *KVServer) HandleApplychSnapshot(raft_type raft.ApplyMsg) {
	if raft_type.SnapshotIndex < kv.lastAppliedIndex {
		return
	}
	snapshot := raft_type.Snapshot
	kv.readPersist(snapshot)
	laneLog.Logger.Infof("server [%d] passive📷 lastAppliedIndex[%d] -> [%d]", kv.me, kv.lastAppliedIndex, raft_type.SnapshotIndex)
	kv.lastAppliedIndex = raft_type.SnapshotIndex
	for {
		select {
		case kv.lastIndexCh <- raft_type.CommandIndex:
		default:
			goto SNAPBREAK
		}
	}
SNAPBREAK:
}

// 主动快照,每一个服务器都在自己log超标的时候启动快照
func (kv *KVServer) checkifNeedSnapshot(spanshotindex int) {
	if kv.maxraftstate == -1 {
		return
	}
	if !kv.rf.IfNeedExceedLog(kv.maxraftstate) {
		return
	} //需要进行快照了

	laneLog.Logger.Infof("server [%d] need snapshot limit[%d] curRaftStatesize[%d] snapshotIndex[%d]", kv.me, kv.maxraftstate, kv.persister.RaftStateSize(), spanshotindex)
	//首先查看一下自己的状态机应用到了那一步

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(kv.duplicateMap); err != nil {
		laneLog.Logger.Fatalf("snapshot duplicateMap encoder fail:%s", err)
	}
	data, err := kv.db.SnapshotData()
	if err != nil {
		laneLog.Logger.Fatalf("database snapshotdata faild:%s", err)
	}
	enc.Encode(data)
	//将状态机传了进去
	kv.rf.Snapshot(spanshotindex, buf.Bytes())

}

// 被动快照
func (kv *KVServer) readPersist(data []byte) {

	if data == nil || len(data) < 1 {
		return
	}
	laneLog.Logger.Infof("server [%d] passive 📷 len of snapshotdate[%d] ", kv.me, len(data))
	laneLog.Logger.Infof("server [%d] before map[%v]", kv.me, kv.db)
	r := bytes.NewBuffer(data)
	d := gob.NewDecoder(r)

	duplicateMap := make(map[int64]duplicateType)
	if err := d.Decode(&duplicateMap); err != nil {
		laneLog.Logger.Fatalf("decode err:%s", err)
	}

	newdb := buntdbx.NewDB()
	dbData := make([]byte, 0)
	err := d.Decode(&dbData)
	if err != nil {
		laneLog.Logger.Fatalln("read persiset err", err)
	}
	newdb.InstallSnapshotData(dbData)
	kv.db = newdb
	kv.duplicateMap = duplicateMap

	laneLog.Logger.Infof("server [%d] after map[%v]", kv.me, kv.db)

}

// the tester calls Kill() when a KVServer instance won't
// be needed again. for your convenience, we supply
// code to set rf.dead (without needing a lock),
// and a killed() method to test rf.dead in
// long-running loops. you can also add your own
// code to Kill(). you're not required to do anything
// about this, but it may be convenient (for example)
// to suppress debug output from a Kill()ed instance.
func (kv *KVServer) Kill() {
	atomic.StoreInt32(&kv.dead, 1)
	kv.rf.Kill()
	// Your code here, if desired.
}

func (kv *KVServer) killed() bool {
	z := atomic.LoadInt32(&kv.dead)
	return z == 1
}

// servers[] contains the ports of the set of
// servers that will cooperate via Raft to
// form the fault-tolerant key/value service.
// me is the index of the current server in servers[].
// the k/v server should store snapshots through the underlying Raft
// implementation, which should call persister.SaveStateAndSnapshot() to
// atomically save the Raft state along with the snapshot.
// the k/v server should snapshot when Raft's saved state exceeds maxraftstate bytes,
// in order to allow Raft to garbage-collect its log. if maxraftstate is -1,
// you don't need to snapshot.
// StartKVServer() must return quickly, so it should start goroutines
// for any long-running work.
func StartKVServer(conf laneConfig.Kvserver, me int, persister *raft.Persister, maxraftstate int) *KVServer {
	// call labgob.Register on structures you want
	// Go's RPC library to marshall/unmarshall.
	var err error
	gob.Register(raft.Op{})
	gob.Register(map[string]string{})
	gob.Register(map[int64]duplicateType{})
	kv := &KVServer{
		me:               me,
		maxraftstate:     maxraftstate,
		persister:        persister,
		applyCh:          make(chan raft.ApplyMsg),
		lastAppliedIndex: 0,
		lastIncludeIndex: 0,
		db:               buntdbx.NewDB(),
		lastIndexCh:      make(chan int),
		duplicateMap:     make(map[int64]duplicateType),
	}

	kv.rf = raft.Make(me, persister, kv.applyCh, conf.Rafts)

	//state machine

	kv.readPersist(persister.ReadSnapshot())
	go kv.HandleApplych()
	// go kv.HandleSnapshot()
	// go kv.handleGetTask()
	// server grpc
	lis, err := net.Listen("tcp", conf.Addr+conf.Port)
	if err != nil {
		laneLog.Logger.Fatalln("error: etcd start faild", err)
	}
	gServer := grpc.NewServer()
	pb.RegisterKvserverServer(gServer, kv)
	go func() {
		if err := gServer.Serve(lis); err != nil {
			laneLog.Logger.Fatalln("failed to serve : ", err.Error())
		}
	}()

	laneLog.Logger.Infoln("etcd serivce is running on addr:", conf.Addr+conf.Port)
	kv.grpc = gServer

	laneLog.Logger.Infof("server [%d] restart", kv.me)
	return kv
}
