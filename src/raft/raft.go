package raft

// 适应kvraft版
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	//	"bytes"

	"bytes"
	"context"
	"encoding/gob"
	"math/rand"
	"net"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Oncelane/laneEtcd/proto/pb"
	"github.com/Oncelane/laneEtcd/src/pkg/laneConfig"
	"github.com/Oncelane/laneEtcd/src/pkg/laneLog"

	"google.golang.org/grpc"
)

const (
	follower = iota
	candidate
	leader
)
const HeartBeatInterval = 100
const TICKMIN = 300
const TICKRANDOM = 300
const LOGINITCAPCITY = 1000
const APPENDENTRIES_TIMES = 0     //对于AE 重传只尝试5次
const APPENDENTRIES_INTERVAL = 20 //对于任何重传AE，间隔20ms重传一次

// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in part 3D you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh, but set CommandValid to false for these
// other uses.
type ApplyMsg struct {
	CommandValid bool
	Command      []byte
	CommandIndex int

	// For 3D:
	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  int
	SnapshotIndex int
}

type LogType struct {
	Term  int
	Value interface{}
}

// A Go object implementing a single Raft peer.
type Raft struct {
	mu        sync.Mutex // Lock to protect shared access to this peer's state
	peers     []*RaftEnd // RPC end points of all peers
	persister *Persister // Object to hold this peer's persisted state
	me        int        // this peer's index into peers[]
	dead      int32      // set by Kill()

	// Your data here (3A, 3B, 3C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.

	//peer state
	state int32

	//persister 持久性
	currentTerm      int
	votedFor         int
	log              []pb.LogType
	lastIncludeIndex int
	lastIncludeTerm  int

	//volatility 易失性
	commitIndex int
	lastApplied int

	//leader volatility
	nextIndex  []int
	matchIndex []int

	//AppendEntris info
	lastHearBeatTime      time.Time
	lastSendHeartbeatTime time.Time

	//leaderId
	leaderId int

	//ApplyCh 提交信息
	applyCh chan ApplyMsg

	//Applyerterm
	applyChTerm chan ApplyMsg

	//snapshotdate
	SnapshotDate []byte

	//Man,what can i say?
	IisBack      bool
	IisBackIndex int
}

// func (rf *Raft) GetDuplicateMap(key int64) (value duplicateType, ok bool) {
// 	value, ok = rf.duplicateMap[key]
// 	return
// }

// func (rf *Raft) SetDuplicateMap(key int64, index int, reply string) {
// 	rf.duplicateMap[key] = duplicateType{
// 		Index: index,
// 		Reply: reply,
// 	}
// }

// func (rf *Raft) DelDuplicateMap(key int64) {
// 	delete(rf.duplicateMap, key)
// }

func (rf *Raft) GetCommitIndex() int {
	return rf.commitIndex + 1
}

func (rf *Raft) Applyer() {
	// for msg := range rf.applyChTerm {
	// 	rf.applyCh <- msg
	// }
	for !rf.killed() {
		select {
		case rf.applyCh <- <-rf.applyChTerm:
			// laneLog.Logger.Infof("Term[%d] [%d] now applyChtemp len=[%d]", rf.currentTerm, rf.me, len(rf.applyChTerm))
		}
	}
}

func (rf *Raft) lastIndex() int {
	return rf.lastIncludeIndex + len(rf.log)
}

func (rf *Raft) lastTerm() int {
	lastLogTerm := rf.lastIncludeTerm
	if len(rf.log) != 0 {
		lastLogTerm = int(rf.log[len(rf.log)-1].Term)
	}
	return lastLogTerm
}

func (rf *Raft) index2LogPos(index int) (pos int) {
	return index - rf.lastIncludeIndex - 1
}

type RequestVoteArgs struct {
	// Your data here (3A, 3B).
	Term         int
	CandidateId  int
	LastLogIndex int
	LastLogTerm  int
}

type RequestVoteReply struct {
	// Your data here (3A).
	Term        int
	VoteGranted bool
}

type AppendEntriesArgs struct {
	Term         int //leader任期
	LeaderId     int //leaderId
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []LogType
	LeaderCommit int
}

type AppendEntriesReply struct {
	Term          int  //接收者任期
	Success       bool //是否接受心跳包
	ConflictIndex int
	ConflictTerm  int
}

type SnapshotInstallArgs struct {
	Term             int
	LeaderId         int
	LastIncludeIndex int
	LastIncludeTerm  int
	Data             []byte
}

type SnapshotInstallreplys struct {
	Term int
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	// Your code here (3A).
	return rf.currentTerm, rf.state == leader
}

func (rf *Raft) GetLeader() bool {
	return rf.state == leader
}

func (rf *Raft) GetTerm() int {
	return rf.currentTerm
}

// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
// before you've implemented snapshots, you should pass nil as the
// second argument to persister.Save().
// after you've implemented snapshots, pass the current snapshot
// (or nil if there's not yet a snapshot).
func (rf *Raft) persist() {
	// Your code here (3C).
	// Example:
	data := rf.persistWithSnapshot()
	rf.persister.Save(data, rf.SnapshotDate)
	// laneLog.Logger.Infof("📦Per Term[%d] [%d] len of persist.Snapshot[%d],len of raft.snapshot[%d]", rf.currentTerm, rf.me, len(rf.persister.snapshot), len(rf.SnapshotDate))
}

func (rf *Raft) persistWithSnapshot() []byte {
	//TODO
	w := new(bytes.Buffer)
	e := gob.NewEncoder(w)
	e.Encode(rf.currentTerm)
	e.Encode(rf.votedFor)
	e.Encode(rf.log)
	e.Encode(rf.lastIncludeIndex)
	e.Encode(rf.lastIncludeTerm)
	// e.Encode(rf.duplicateMap)
	raftstate := w.Bytes()
	return raftstate
}

func (rf *Raft) readPersist(data []byte) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (3C).
	// Example:
	r := bytes.NewBuffer(data)
	d := gob.NewDecoder(r)

	var currentTerm int
	var votedFor int
	var log []pb.LogType
	var lastIncludeIndex int
	var lastIncludeTerm int

	d.Decode(&currentTerm)
	d.Decode(&votedFor)
	d.Decode(&log)
	d.Decode(&lastIncludeIndex)
	d.Decode(&lastIncludeTerm)

	rf.currentTerm = currentTerm
	rf.votedFor = votedFor
	rf.log = append(rf.log, log...)
	rf.lastIncludeIndex = lastIncludeIndex
	rf.lastIncludeTerm = lastIncludeTerm

}

// example RequestVote RPC handler.

// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.

func (rf *Raft) CopyEntries(args *pb.AppendEntriesArgs) {
	// logchange := false
	for i := 0; i < len(args.Entries); i++ {
		rfIndex := i + int(args.PrevLogIndex) + 1
		logPos := rf.index2LogPos(rfIndex)
		if rfIndex > rf.lastIndex() { //超出原本log长度了
			// rf.log = append(rf.log, args.Entries[i:]...)
			for j := i; j < len(args.Entries); j++ {
				rf.log = append(rf.log, *args.Entries[j])
			}
			rf.persist()
			// logchange = true
			break
		} else if rf.log[logPos].Term != args.Entries[i].Term { //有脏东西
			rf.log = rf.log[:logPos] //删除脏数据
			//一口气复制完
			for j := i; j < len(args.Entries); j++ {
				rf.log = append(rf.log, *args.Entries[j])
			}
			rf.persist()
			// logchange = true
			break
		}
	}
	//用于debug
	// if logchange {
	// 	laneLog.Logger.Infof("💖Rev Term[%d] [%d] Copy: Len -> [%d] ", rf.currentTerm, rf.me, len(rf.log))

	// 	laneLog.Logger.Infof("Term[%d] [%d] after copy:", rf.currentTerm, rf.me)
	// 	i := len(rf.log) - 10
	// 	if i < 0 {
	// 		i = 0
	// 	}
	// 	// for ; i < len(rf.log); i++ {
	// 	// 	laneLog.Logger.Infof("Term[%d] [%d] index[%d] log[%v]", rf.currentTerm, rf.me, i+rf.lastIncludeIndex+1, rf.log[i].Value)
	// 	// }

	// }
	var min = -1
	if args.LeaderCommit > int64(rf.lastIndex()) {
		min = rf.lastIndex()
	} else {
		min = int(args.LeaderCommit)
	}
	if rf.commitIndex < min {
		// laneLog.Logger.Infof("COMIT Term[%d] [%d] CommitIndex: [%d] -> [%d]", rf.currentTerm, rf.me, rf.commitIndex, min)
		rf.commitIndex = min
		rf.undateLastApplied()
	}

}
func (rf *Raft) RequestVote(_ context.Context, args *pb.RequestVoteArgs) (reply *pb.RequestVoteReply, err error) {
	reply = new(pb.RequestVoteReply)
	// laneLog.Logger.Panicln("check for args", args)
	rf.mu.Lock()
	defer rf.mu.Unlock()
	defer laneLog.Logger.Infof("🎫Rec Term[%d] [%d]reply [%d]Vote finish", rf.currentTerm, rf.me, args.CandidateId)
	// defer func() { //defer最后运行
	// 	if err := recover(); err != nil {
	// 		laneLog.Logger.Errorf("程序报错了，错误信息为=%s\n", err)
	// 	}
	// }()
	reply.VoteGranted = false
	reply.Term = int64(rf.currentTerm)

	if args.Term < int64(rf.currentTerm) {
		return
	}
	if rf.currentTerm < int(args.Term) {
		rf.currentTerm = int(args.Term)
		rf.state = follower
		rf.votedFor = -1
		rf.leaderId = -1
		rf.persist()
	}
	if rf.votedFor == -1 || rf.votedFor == int(args.CandidateId) {
		lastLogTerm := rf.lastTerm()
		if args.LastLogTerm > int64(lastLogTerm) || (args.LastLogTerm == int64(lastLogTerm) && args.LastLogIndex >= int64(rf.lastIndex())) {
			rf.votedFor = int(args.CandidateId)
			reply.VoteGranted = true
			rf.lastHearBeatTime = time.Now()
			laneLog.Logger.Infof("🎫Rec Term[%d] [%d] -> [%d]", rf.currentTerm, rf.me, args.CandidateId)
		}
	}
	rf.persist()
	return
}
func (rf *Raft) AppendEntries(_ context.Context, args *pb.AppendEntriesArgs) (reply *pb.AppendEntriesReply, err error) {
	reply = new(pb.AppendEntriesReply)
	//需要补充判断条件
	rf.mu.Lock()
	defer rf.mu.Unlock()
	reply.Success = false
	reply.Term = int64(rf.currentTerm)
	reply.ConflictIndex = -1
	reply.ConflictTerm = -1

	if rf.currentTerm > int(args.Term) {
		laneLog.Logger.Infof("💔Rec Term[%d] [%d] Reject Leader[%d]Term[%d][too OLE]", rf.currentTerm, rf.me, args.LeaderId, int(args.Term))
		return
	}
	// rf.tryChangeToFollower(int(args.Term))
	if rf.currentTerm < int(args.Term) {
		rf.currentTerm = int(args.Term)
		rf.state = follower
		rf.votedFor = -1
		rf.persist()
	}
	if rf.currentTerm == int(args.Term) && atomic.LoadInt32(&rf.state) == candidate {
		rf.state = follower
		rf.votedFor = -1
		rf.persist()
	}
	rf.lastHearBeatTime = time.Now()
	rf.leaderId = int(args.LeaderId)
	// laneLog.Logger.Infof("💖Rec Term[%d] [%d] Receive: LeaderId[%d]Term[%d] PreLogIndex[%d] PrevLogTerm[%d] LeaderCommit[%d] Entries[%v] len[%d]", rf.currentTerm, rf.me, args.LeaderId, args.Term, args.PrevLogIndex, args.PrevLogTerm, args.LeaderCommit, args.Entries, len(args.Entries))
	//新判断
	if args.PrevLogIndex < int64(rf.lastIncludeIndex) { // index在快照范围内，那么
		reply.ConflictIndex = 0
		laneLog.Logger.Infof("💔Rec Term[%d] [%d] Reject for args.PrevLogIndex[%d] < rf.lastIncludeIndex[%d]", rf.currentTerm, rf.me, args.PrevLogIndex, rf.lastIncludeIndex)
		return
	} else if args.PrevLogIndex == int64(rf.lastIncludeIndex) {
		if args.PrevLogTerm != int64(rf.lastIncludeTerm) {
			reply.ConflictIndex = 0
			laneLog.Logger.Infof("💔Rec Term[%d] [%d] Reject for args.PrevLogTermk[%d] != rf.lastIncludeTerm[%d]", rf.currentTerm, rf.me, args.PrevLogIndex, rf.lastIncludeIndex)
			return
		}
	} else { //index在快照范围外，那么正常走日志覆盖逻辑
		if rf.lastIndex() < int(args.PrevLogIndex) {
			reply.ConflictIndex = int64(rf.lastIndex())
			laneLog.Logger.Infof("💔Rec Term[%d] [%d] Reject:PreLogIndex[%d] Out of Len ->[%d]", rf.currentTerm, rf.me, args.PrevLogIndex, rf.lastIndex())
			return
		}

		if args.PrevLogIndex >= 0 && rf.log[rf.index2LogPos(int(args.PrevLogIndex))].Term != int64(args.PrevLogTerm) {
			reply.ConflictTerm = int64(rf.log[rf.index2LogPos(int(args.PrevLogIndex))].Term)
			for index := rf.lastIncludeIndex + 1; index <= int(args.PrevLogIndex); index++ { // 找到冲突term的首次出现位置，最差就是PrevLogIndex
				if rf.log[rf.index2LogPos(int(args.PrevLogIndex))].Term == int64(reply.ConflictTerm) {
					reply.ConflictIndex = int64(index)
					break
				}
			}
			laneLog.Logger.Infof("💔Rev Term[%d] [%d] Reject :PreLogTerm Not Match [%d] != [%d]", rf.currentTerm, rf.me, rf.log[rf.index2LogPos(int(args.PrevLogIndex))].Term, args.PrevLogTerm)
			return
		}
	}
	//保存日志
	rf.CopyEntries(args)
	reply.Success = true
	rf.persist()
	return
}

func (rf *Raft) SnapshotInstall(_ context.Context, args *pb.SnapshotInstallArgs) (reply *pb.SnapshotInstallReply, err error) {
	reply = new(pb.SnapshotInstallReply)
	rf.mu.Lock()
	defer rf.mu.Unlock()
	laneLog.Logger.Infof("SNAPS Term[%d] [%d] Receiv📷 from[%d] lastIncludeIndex[%d] lastIncludeTerm[%d]", rf.currentTerm, rf.me, args.LeaderId, args.LastIncludeIndex, args.LastIncludeTerm)

	reply.Term = int64(rf.currentTerm)

	if args.Term < int64(rf.currentTerm) {
		laneLog.Logger.Infof("SNAPS Term[%d] [%d] reject📷 for it's Term[%d] [too old]", rf.currentTerm, rf.me, args.Term)
		return
	}

	if args.Term > int64(rf.currentTerm) {
		rf.currentTerm = int(args.Term)
		rf.state = follower
		rf.votedFor = -1
		rf.persist()
	}

	rf.leaderId = int(args.LeaderId)
	rf.lastHearBeatTime = time.Now()

	if args.LastIncludeIndex <= int64(rf.lastIncludeIndex) {
		return
	} else {
		if args.LastIncludeIndex < int64(rf.lastIndex()) {
			if rf.log[rf.index2LogPos(int(args.LastIncludeIndex))].Term != int64(args.LastIncludeTerm) {
				rf.log = make([]pb.LogType, 0)
			} else {
				leftLog := make([]pb.LogType, rf.lastIndex()-int(args.LastIncludeIndex))
				copy(leftLog, rf.log[rf.index2LogPos(int(args.LastIncludeIndex)+1):])
				rf.log = leftLog
				// rf.log = rf.log[rf.index2LogPos(args.LastIncludeIndex+1):]
			}
		} else {
			rf.log = make([]pb.LogType, 0)
		}
	}
	laneLog.Logger.Infof("SNAPS Term[%d] [%d] Accept📷 Now it's lastIncludeIndex [%d] -> [%d] lastIncludeTerm [%d] -> [%d]", rf.currentTerm, rf.me, rf.lastIncludeIndex, args.LastIncludeIndex, rf.lastIncludeTerm, args.LastIncludeTerm)
	laneLog.Logger.Infof("snaps Term[%d] [%d] after snapshot log:", rf.currentTerm, rf.me)

	rf.lastIncludeIndex = int(args.LastIncludeIndex)
	rf.lastIncludeTerm = int(args.LastIncludeTerm)

	i := len(rf.log) - 10
	if i < 0 {
		i = 0
	}
	for ; i < len(rf.log); i++ {
		laneLog.Logger.Infof("Term[%d] [%d] index[%d] value[term:%v data:%v]", rf.currentTerm, rf.me, i+rf.lastIncludeIndex+1, rf.log[i].Term, rf.log[i].Value)
	}
	spanshootLength := rf.persister.SnapshotSize()
	laneLog.Logger.Infof("📷Cmi Term[%d] [%d] 📦Save snapshot to application[%d] (Receive from leader)", rf.currentTerm, rf.me, spanshootLength)
	//snapshot提交给应用层
	applyMsg := ApplyMsg{
		SnapshotValid: true,
		Snapshot:      args.Data,
		SnapshotIndex: rf.lastIncludeIndex + 1, //记得这里有个坑
		SnapshotTerm:  rf.lastIncludeTerm,
	}
	//快照提交给了application
	rf.lastApplied = rf.lastIncludeIndex
	laneLog.Logger.Infof("📷Cmi Term[%d] [%d] Ready to commit snapshot snapshotIndex[%d] snapshotTerm[%d]", rf.currentTerm, rf.me, rf.lastIncludeIndex, rf.lastIncludeTerm)
	rf.mu.Unlock()
	rf.applyChTerm <- applyMsg
	rf.mu.Lock()
	//持久化快照
	rf.SnapshotDate = args.Data
	rf.persister.Save(rf.persistWithSnapshot(), args.Data)
	laneLog.Logger.Infof("📷Cmi Term[%d] [%d] Done Success to comit snapshot snapshotIndex[%d] snapshotTerm[%d]", rf.currentTerm, rf.me, rf.lastIncludeIndex, rf.lastIncludeTerm)
	return
}

func (rf *Raft) installSnapshotToApplication() {
	// rf.snapshotXapplych.Lock()
	// defer rf.snapshotXapplych.Unlock()
	//snapshot提交给应用层
	applyMsg := &ApplyMsg{
		SnapshotValid: true,
		Snapshot:      rf.persister.ReadSnapshot(),
		SnapshotIndex: rf.lastIncludeIndex + 1, //记得这里有个坑
		SnapshotTerm:  rf.lastIncludeTerm,
	}
	//快照提交给了application
	snapshotLength := rf.persister.SnapshotSize()
	if snapshotLength < 1 {
		laneLog.Logger.Infof("📷Cmi Term[%d] [%d] Snapshotlen[%d] No need to commit snapshotIndex[%d] snapshotTerm[%d] ", rf.currentTerm, rf.me, snapshotLength, rf.lastIncludeIndex, rf.lastIncludeTerm)
		return
	}
	rf.SnapshotDate = rf.persister.ReadSnapshot()
	rf.lastApplied = rf.lastIncludeIndex
	laneLog.Logger.Infof("📷Cmi Term[%d] [%d] Ready to commit snapshot snapshotIndex[%d] snapshotTerm[%d]", rf.currentTerm, rf.me, rf.lastIncludeIndex, rf.lastIncludeTerm)
	rf.applyChTerm <- *applyMsg
	laneLog.Logger.Infof("📷Cmi Term[%d] [%d] Done Success to comit snapshot snapshotIndex[%d] snapshotTerm[%d]", rf.currentTerm, rf.me, rf.lastIncludeIndex, rf.lastIncludeTerm)
}

func (rf *Raft) sendInstallSnapshot(server int, args *pb.SnapshotInstallArgs) (reply *pb.SnapshotInstallReply, ok bool) {
	reply, err := rf.peers[server].conn.SnapshotInstall(context.Background(), args)
	return reply, err == nil
}

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// Your code here (3D).
	// laneLog.Logger.Infof("SNAPS Term[%d] [%d] 📷Snapshot ask to snap Index[%d] Raft log Len:[%d]", rf.currentTerm, rf.me, index-1, len(rf.log))
	// laneLog.Logger.Infof("SNAPS Term[%d] [%d] Wait for the lock🤨", rf.currentTerm, rf.me)
	rf.mu.Lock()
	// laneLog.Logger.Infof("SNAPS Term[%d] [%d] Get the lock🔐", rf.currentTerm, rf.me)
	// defer laneLog.Logger.Infof("SNAPS Term[%d] [%d] Unlock the lock🔓", rf.currentTerm, rf.me)
	defer rf.mu.Unlock()

	index -= 1
	if index <= rf.lastIncludeIndex {
		return
	}
	compactLoglen := index - rf.lastIncludeIndex
	// laneLog.Logger.Infof("SNAPS Term[%d] [%d] After📷,lastIncludeIndex[%d]->[%d] lastIncludeTerm[%d]->[%d] len of Log->[%d]", rf.currentTerm, rf.me, rf.lastIncludeIndex, index, rf.lastIncludeTerm, rf.log[rf.index2LogPos(index)].Term, len(rf.log)-compactLoglen)

	rf.lastIncludeTerm = int(rf.log[rf.index2LogPos(index)].Term)
	rf.lastIncludeIndex = index

	//压缩日志
	afterLog := make([]pb.LogType, len(rf.log)-compactLoglen)
	copy(afterLog, rf.log[compactLoglen:])
	rf.log = afterLog
	//把snapshot和raftstate持久化
	rf.SnapshotDate = snapshot
	rf.persister.Save(rf.persistWithSnapshot(), snapshot)
	// laneLog.Logger.Infof("📷Cmi Term[%d] [%d] 📦Save snapshot to application[%d] (Receive from up Application)", rf.currentTerm, rf.me, rf.persister.SnapshotSize())
}

const (
	AEtry = iota
	AElostRPC
	AERejectRPC
)

// 外部调用负责上锁，此处不上锁
func (rf *Raft) updateCommitIndex() {
	//从matchIndex寻找一个大多数服务器认同的N
	if rf.state == leader {
		matchIndex := make([]int, 0)
		matchIndex = append(matchIndex, rf.lastIndex())
		for i := range rf.peers {
			if i != rf.me {
				matchIndex = append(matchIndex, rf.matchIndex[i])
			}
		}

		sort.Ints(matchIndex)

		lenMat := len(matchIndex) //2 两台follower
		N := matchIndex[lenMat/2] //1
		if N > rf.commitIndex && (N <= rf.lastIncludeIndex || rf.currentTerm == int(rf.log[rf.index2LogPos(N)].Term)) {
			// laneLog.Logger.Infof("COMIT Term[%d] [%d] It's matchIndex = %v", rf.currentTerm, rf.me, matchIndex)
			// laneLog.Logger.Infof("COMIT Term[%d] [%d] commitIndex [%d] -> [%d] (leader action)", rf.currentTerm, rf.me, rf.commitIndex, N)
			rf.IisBackIndex = N
			rf.commitIndex = N
			rf.undateLastApplied()
		}
	}
}

// 修改rf.lastApplied
func (rf *Raft) undateLastApplied() {
	// laneLog.Logger.Infof("APPLY Term[%d] [%d] Wait for the lock🔐", rf.currentTerm, rf.me)
	// laneLog.Logger.Infof("APPLY Term[%d] [%d] Hode the lock🔐", rf.currentTerm, rf.me)
	for rf.lastApplied < rf.commitIndex {
		rf.lastApplied += 1
		index := rf.index2LogPos(rf.lastApplied)
		if index <= -1 || index >= len(rf.log) {
			rf.lastApplied = rf.lastIncludeIndex
			laneLog.Logger.Errorf("ERROR? 👿 [%d]Ready to apply index[%d] But index out of Len of log, lastApplied[%d] commitIndex[%d] lastIncludeIndex[%d] logLen:%d", rf.me, index, rf.lastApplied, rf.commitIndex, rf.lastIncludeIndex, len(rf.log))
			rf.mu.Unlock()
			return
		}
		// laneLog.Logger.Infof("APPLY Term[%d] [%d] -> LOG [%d] value:[%d]", rf.currentTerm, rf.me, rf.lastApplied, rf.log[index].Value)

		ApplyMsg := ApplyMsg{
			CommandValid: true,
			Command:      rf.log[index].Value,
			CommandIndex: rf.lastApplied + 1,
		}
		// laneLog.Logger.Infof("APPLY Term[%d] [%d] Unlock the lock🔐 For Start applyerCh <- len[%d]", rf.currentTerm, rf.me, len(rf.applyChTerm))
		rf.applyChTerm <- ApplyMsg
		if rf.IisBackIndex == rf.lastApplied {
			// laneLog.Logger.Infof("Term [%d] [%d] iisback = true iisbackIndex =[%d]", rf.currentTerm, rf.me, rf.IisBackIndex)
			rf.IisBack = true
		}
	}

}

// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.

func (rf *Raft) Start(command []byte) (int, int, bool) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	index := rf.lastIndex() + 2
	term := rf.currentTerm
	isLeader := rf.state == leader

	// Your code here (3B).
	if !isLeader {
		return index, term, isLeader
	}

	//添加条目到本地
	rf.log = append(rf.log, pb.LogType{
		Term:  int64(rf.currentTerm),
		Value: command,
	})
	rf.persist()

	// laneLog.Logger.Infof("CLIENT📨 Term[%d] [%d] Receive [%v] logIndex[%d](leader action)\n", rf.currentTerm, rf.me, command, index-1)
	return index, term, isLeader
}

// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

func (rf *Raft) sendRequestVote2(server int, args *pb.RequestVoteArgs) (reply *pb.RequestVoteReply, ok bool) {
	reply, err := rf.peers[server].conn.RequestVote(context.Background(), args)
	return reply, err == nil
}

func (rf *Raft) electionLoop() {
	for !rf.killed() {
		time.Sleep(time.Microsecond * 50)
		func() {
			rf.mu.Lock()
			defer rf.mu.Unlock()

			timeCount := time.Since(rf.lastHearBeatTime).Milliseconds()
			ms := TICKMIN + rand.Int63()%TICKRANDOM
			if rf.state == follower {
				if timeCount >= ms {
					laneLog.Logger.Infof("❗Term[%d] [%d] Follower -> Candidate", rf.currentTerm, rf.me)
					rf.state = candidate
					rf.leaderId = -1
				}
			}
			if rf.state == candidate && timeCount >= ms {
				rf.lastHearBeatTime = time.Now()
				rf.leaderId = -1
				rf.currentTerm += 1
				rf.votedFor = rf.me
				rf.persist()

				//请求投票
				args := pb.RequestVoteArgs{
					Term:         int64(rf.currentTerm),
					CandidateId:  int64(rf.me),
					LastLogIndex: int64(rf.lastIndex()),
					LastLogTerm:  int64(rf.lastTerm()),
				}
				rf.mu.Unlock()

				type VoteResult struct {
					raftId int
					resp   *pb.RequestVoteReply
				}
				voteCount := 1
				finishCount := 1
				VoteResultChan := make(chan *VoteResult, len(rf.peers))
				for peerId := 0; peerId < len(rf.peers) && !rf.killed(); peerId++ {
					go func(server int) {
						if server == rf.me {
							return
						}
						if resp, ok := rf.sendRequestVote2(server, &args); ok {

							if resp.VoteGranted {
								laneLog.Logger.Infof("🎫Get Term[%d] [%d]Candidate 🥰receive a voteRPC reply from [%d] ,voteGranted Yes", rf.currentTerm, rf.me, server)
							} else {
								laneLog.Logger.Infof("🎫Get Term[%d] [%d]Candidate 🥰receive a voteRPC reply from [%d] ,voteGranted No", rf.currentTerm, rf.me, server)
							}
							VoteResultChan <- &VoteResult{raftId: server, resp: resp}

						} else {
							// laneLog.Logger.Infof("🎫Get Term[%d] [%d]Candidate 🥲Do not get voteRPC reply from [%d] ,voteGranted Nil", rf.currentTerm, rf.me, server)
							VoteResultChan <- &VoteResult{raftId: server, resp: nil}
						}

					}(peerId)
				}
				maxTerm := 0
				for !rf.killed() {
					select {
					case VoteResult := <-VoteResultChan:
						finishCount += 1
						if VoteResult.resp != nil {
							if VoteResult.resp.VoteGranted {
								voteCount += 1
							}
							if int(VoteResult.resp.Term) > maxTerm {
								maxTerm = int(VoteResult.resp.Term)
							}
						}

						if finishCount == len(rf.peers) || voteCount > len(rf.peers)/2 {
							goto VOTE_END
						}
					case <-time.After(time.Duration(TICKMIN+rand.Int63()%TICKRANDOM) * time.Millisecond):
						laneLog.Logger.Infof("🎫Get Term[%d] [%d]Candidate Fail🥲 election time out", rf.currentTerm, rf.me)
						goto VOTE_END
					}
				}
			VOTE_END:
				rf.mu.Lock()

				if rf.state != candidate {
					return
				}

				if maxTerm > rf.currentTerm {
					rf.state = follower
					rf.leaderId = -1
					rf.currentTerm = maxTerm
					rf.votedFor = -1
					rf.persist()
					return
				}

				if voteCount > len(rf.peers)/2 {
					rf.IisBack = false
					rf.state = leader
					rf.leaderId = rf.me
					rf.nextIndex = make([]int, len(rf.peers))
					for i := 0; i < len(rf.peers); i++ {
						rf.nextIndex[i] = rf.lastIndex() + 1
					}
					rf.matchIndex = make([]int, len(rf.peers))
					for i := 0; i < len(rf.peers); i++ {
						rf.matchIndex[i] = -1
					}
					rf.updateCommitIndex()
					laneLog.Logger.Infof("❗ Term[%d] [%d]candidate -> leader", rf.currentTerm, rf.me)
					rf.lastSendHeartbeatTime = time.Now().Add(-time.Millisecond * 2 * HeartBeatInterval)
					// data, _ := json.Marshal(Op{
					// 	OpType: int32(pb.OpType_EmptyT),
					// })
					b := new(bytes.Buffer)
					e := gob.NewEncoder(b)
					e.Encode(Op{
						OpType: int32(pb.OpType_EmptyT),
					})

					rf.mu.Unlock()
					rf.Start(b.Bytes())
					rf.mu.Lock()
					return
				}
				// laneLog.Logger.Infof("🎫Rec Term[%d] [%d]candidate Fail to get majority Vote", rf.currentTerm, rf.me)
			}

		}()
	}
}

const (
	AEresult_Accept      = iota //接受日志
	AEresult_Reject             //不接受日志
	AEresult_StopSending        //我任期比你大！
	AEresult_Ignore             //上个任期或更久前发送的心跳的响应
	AEresult_Lost               //直到超时，对方没收到心跳包/己方没收到响应
)

func (rf *Raft) SendAppendEntriesToPeerId(server int, applychreply *chan int) {
	rf.mu.Lock()
	if rf.state != leader {
		rf.mu.Unlock()
		if applychreply != nil {
			*applychreply <- AEresult_StopSending
		}
		return
	}
	args := &pb.AppendEntriesArgs{}
	reply := &pb.AppendEntriesReply{}
	*args = pb.AppendEntriesArgs{
		Term:         int64(rf.currentTerm),
		LeaderId:     int64(rf.me),
		PrevLogIndex: int64(rf.nextIndex[server] - 1),
		PrevLogTerm:  -1,
		Entries:      make([]*pb.LogType, 0),
		LeaderCommit: int64(rf.commitIndex),
	}
	if int(args.PrevLogIndex) == rf.lastIncludeIndex {
		args.PrevLogTerm = int64(rf.lastIncludeTerm)
	}

	if rf.index2LogPos(int(args.PrevLogIndex)) >= 0 && rf.index2LogPos(int(args.PrevLogIndex)) < len(rf.log) { //有PrevIndex
		args.PrevLogTerm = int64(rf.log[rf.index2LogPos(int(args.PrevLogIndex))].Term)
	}
	if rf.index2LogPos(rf.nextIndex[server]) >= 0 && rf.index2LogPos(rf.nextIndex[server]) < len(rf.log) { //有nextIndex
		entrys := rf.log[rf.index2LogPos(rf.nextIndex[server]):]
		// args.Entries = make([]*pb.LogType, len(rf.log)-rf.index2LogPos(rf.nextIndex[server]))
		// copy(entrys, rf.log[rf.index2LogPos(rf.nextIndex[server]):])
		// for i := rf.index2LogPos(rf.nextIndex[server]); i < len(rf.log); i++ {
		// 	args.Entries = append(args.Entries, &entrys[i])
		// }
		for i := range entrys {
			args.Entries = append(args.Entries, &entrys[i])
		}
		// args.Entries = append(args.Entries, rf.log[rf.index2LogPos(rf.nextIndex[server]):]...)
	}
	rf.mu.Unlock()

	reply, err := rf.peers[server].conn.AppendEntries(context.Background(), args)

	rf.mu.Lock()
	defer rf.mu.Unlock()
	if err == nil {
		if args.Term != int64(rf.currentTerm) {
			laneLog.Logger.Infof("💔Rec Term[%d] [%d] Receive Send.Term[%d][too OLD]", rf.currentTerm, rf.me, args.Term)
			if applychreply != nil {
				*applychreply <- AEresult_Ignore
			}
			return
		}
		if rf.currentTerm < int(reply.Term) {
			rf.votedFor = -1
			rf.state = follower
			rf.currentTerm = int(reply.Term)
			rf.leaderId = -1
			rf.persist()
			laneLog.Logger.Infof("💔Rec Term[%d] [%d] Receive Discover newer Term[%d]", rf.currentTerm, rf.me, reply.Term)
			if applychreply != nil {
				*applychreply <- AEresult_StopSending
			}
			return
		}
		if reply.Success {
			rf.nextIndex[server] = int(args.PrevLogIndex) + len(args.Entries) + 1
			rf.matchIndex[server] = rf.nextIndex[server] - 1 //做闭右开，因此curLatestIndex指向的是最后一个发送的log的下一位可能为空
			rf.updateCommitIndex()
			if applychreply != nil {
				*applychreply <- AEresult_Accept
			}
			return
		} else {

			if reply.ConflictTerm != -1 {
				searchIndex := -1
				for i := int(args.PrevLogIndex); i > rf.lastIncludeIndex; i-- {
					if rf.log[rf.index2LogPos(i)].Term == reply.ConflictTerm {
						searchIndex = i
					}
				}
				if searchIndex != -1 {
					rf.nextIndex[server] = searchIndex + 1
				} else {
					rf.nextIndex[server] = int(reply.ConflictIndex)
				}
			} else {
				rf.nextIndex[server] = int(reply.ConflictIndex) + 1
			}
			if applychreply != nil {
				*applychreply <- AEresult_Reject
			}
			return
		}
	}
	if applychreply != nil {
		*applychreply <- AEresult_Lost
	}
}

func (rf *Raft) appendEntriesLoop() {
	for !rf.killed() {
		time.Sleep(time.Millisecond * 50)
		func() {
			rf.mu.Lock()
			defer rf.mu.Unlock()
			if rf.state != leader {
				return
			}
			countTime := time.Since(rf.lastSendHeartbeatTime).Milliseconds()
			if countTime < HeartBeatInterval {
				return
			}
			rf.lastSendHeartbeatTime = time.Now()
			//发送心跳
			rf.SendAppendEntriesToAll()
		}()
	}
}

func (rf *Raft) SendAppendEntriesToAll() {
	for i := range rf.peers {
		if i == rf.me {
			continue
		}
		if rf.killed() {
			return
		}
		if rf.nextIndex[i] <= rf.lastIncludeIndex {
			go rf.sendInstallSnapshotToPeerId(i)
		} else {
			go rf.SendAppendEntriesToPeerId(i, nil)
		}

	}
}

func (rf *Raft) SendOnlyAppendEntriesToAll() {
	for i := range rf.peers {
		if i == rf.me {
			continue
		}
		if rf.killed() {
			return
		}
		go rf.SendAppendEntriesToPeerId(i, nil)

	}
}

func (rf *Raft) sendInstallSnapshotToPeerId(server int) {
	// laneLog.Logger.Infof("SNAPS Term[%d] [%d] goSend📷Wait for a lock🤨 to [%d],", rf.currentTerm, rf.me, server)
	rf.mu.Lock()
	args := &pb.SnapshotInstallArgs{}
	args.Term = int64(rf.currentTerm)
	args.LeaderId = int64(rf.me)
	args.LastIncludeIndex = int64(rf.lastIncludeIndex)
	args.LastIncludeTerm = int64(rf.lastIncludeTerm)
	args.Data = rf.SnapshotDate
	laneLog.Logger.Infof("SNAPS Term[%d] [%d] goSend📷 to [%d] args.LastIncludeIndex[%d],args.LastIncludeTerm[%d],len of snapshot[%d],", rf.currentTerm, rf.me, server, args.LastIncludeIndex, args.LastIncludeTerm, len(args.Data))
	rf.mu.Unlock()
	go func(args *pb.SnapshotInstallArgs) {
		reply, ok := rf.sendInstallSnapshot(server, args)
		if ok {
			rf.mu.Lock()
			defer rf.mu.Unlock()
			if int64(rf.currentTerm) != args.Term {
				return
			}
			if reply.Term > int64(rf.currentTerm) {
				rf.state = follower
				rf.leaderId = -1
				rf.currentTerm = int(reply.Term)
				rf.votedFor = -1
				rf.persist()
				return
			}
			laneLog.Logger.Infof("SNAPS Term[%d] [%d] leader success to Send a 📷 to [%d] nextIndex for it [%d] -> [%d] matchIndex [%d] -> [%d]", rf.currentTerm, rf.me, server, rf.nextIndex[server], rf.lastIndex()+1, rf.matchIndex[server], args.LastIncludeIndex)
			rf.nextIndex[server] = rf.lastIndex() + 1
			rf.matchIndex[server] = int(args.LastIncludeIndex)
			rf.updateCommitIndex()
		}
	}(args)
}

func (rf *Raft) IfNeedExceedLog(logSize int) bool {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if rf.persister.RaftStateSize() >= logSize {
		return true
	} else {
		return false
	}
}

// -------------new-method-to-adjust-client----------
func (rf *Raft) GetleaderId() int {
	return rf.leaderId
}

func (rf *Raft) CheckIfDepose() (ret bool) {
	applychreply := make(chan int)
	for i := range rf.peers {
		if i == rf.me {
			continue
		}
		go rf.SendAppendEntriesToPeerId(i, &applychreply)
	}
	//算上自己也是一个AE响应
	countAEreply := 1
	countAEreplyTotal := 1
	stopFlag := false
	for !rf.killed() {
		AEreply := <-applychreply
		switch AEreply {
		case AEresult_Accept:
			countAEreply++
			countAEreplyTotal++
		case AEresult_StopSending:
			//不能提前return 否则通道关闭就产生阻塞了
			//此情况不管收到多少countAEreply都不能接受，需要选举新的leader
			stopFlag = true
			countAEreplyTotal++
		default:
			countAEreplyTotal++
		}
		if countAEreplyTotal == len(rf.peers) {
			if stopFlag {
				return true
			}
			if countAEreply > countAEreplyTotal/2 {
				return false
			} else {
				return true
			}
		}
	}

	return true
}

// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
func Make(me int,
	persister *Persister, applyCh chan ApplyMsg, conf laneConfig.RaftEnds) *Raft {
	laneLog.Logger.Debugf("raft[%d] start by conf", me, conf)
	rf := &Raft{}

	rf.persister = persister
	rf.me = me

	//state
	rf.state = follower
	rf.leaderId = -1
	//persister
	rf.currentTerm = 0
	rf.votedFor = -1
	rf.log = make([]pb.LogType, 0, LOGINITCAPCITY)
	rf.lastIncludeIndex = -1
	rf.lastIncludeTerm = -1

	//volatility
	rf.commitIndex = -1
	rf.lastApplied = -1

	//leader volatility
	rf.nextIndex = make([]int, len(rf.peers))
	rf.matchIndex = make([]int, len(rf.peers))
	for i := range rf.matchIndex {
		rf.matchIndex[i] = -1
	}
	rf.updateCommitIndex()
	//heartBeat
	rf.lastHearBeatTime = time.Now()
	rf.lastSendHeartbeatTime = time.Now()
	//leaderId
	rf.leaderId = -1

	//ApplyCh
	rf.applyCh = applyCh
	rf.SnapshotDate = nil
	//
	rf.applyChTerm = make(chan ApplyMsg, 1000)

	//

	rf.IisBack = false
	rf.IisBackIndex = -1
	// Your initialization code here (3A, 3B, 3C).
	laneLog.Logger.Infof("RESTA Term[%d] [%d] Restart😎", rf.currentTerm, rf.me)
	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())
	// 向application层安装快照
	rf.installSnapshotToApplication()
	// start ticker goroutine to start elections
	// go rf.mainLoop2()

	rf.StartRaft(conf.Endpoints[me])
	servers := WaitConnect(conf)
	rf.peers = servers

	go rf.Applyer()
	go rf.electionLoop()
	go rf.appendEntriesLoop()
	// go rf.undateLastApplied()
	// go rf.updateCommitIndex()

	return rf
}

func (rt *Raft) StartRaft(conf laneConfig.RaftEnd) {
	// server grpc
	lis, err := net.Listen("tcp", conf.Addr+conf.Port)
	if err != nil {
		laneLog.Logger.Fatalln("error: etcd start faild", err)
	}
	gServer := grpc.NewServer()
	pb.RegisterRaftServer(gServer, rt)
	go func() {
		if err := gServer.Serve(lis); err != nil {
			laneLog.Logger.Fatalln("failed to serve : ", err.Error())
		}
	}()
}

func WaitConnect(conf laneConfig.RaftEnds) []*RaftEnd {
	laneLog.Logger.Infoln("start wating...")
	var wait sync.WaitGroup
	servers := make([]*RaftEnd, len(conf.Endpoints))
	wait.Add(len(servers) - 1)
	for i := range conf.Endpoints {
		if i == conf.Me {
			continue
		}

		go func(other int, conf laneConfig.RaftEnd) {
			defer wait.Done()
			for {
				r := NewRaftClient(conf)
				if r != nil {
					servers[other] = r
					break
				}
				time.Sleep(time.Millisecond * 500)
			}
		}(i, conf.Endpoints[i])
	}
	wait.Wait()
	laneLog.Logger.Infof("🦖 All %d Connetct", len(conf.Endpoints))
	return servers
}
