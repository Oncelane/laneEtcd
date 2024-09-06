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
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
	"time"
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
	Command      interface{}
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
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	// Your data here (3A, 3B, 3C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.

	//peer state
	state int32

	//persister 持久性
	currentTerm      int
	votedFor         int
	log              []LogType
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
			DPrintf("Term[%d] [%d] now applyChtemp len=[%d]", rf.currentTerm, rf.me, len(rf.applyChTerm))
		}
	}
}

func (rf *Raft) lastIndex() int {
	return rf.lastIncludeIndex + len(rf.log)
}

func (rf *Raft) lastTerm() int {
	lastLogTerm := rf.lastIncludeTerm
	if len(rf.log) != 0 {
		lastLogTerm = rf.log[len(rf.log)-1].Term
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
	// DPrintf("📦Per Term[%d] [%d] len of persist.Snapshot[%d],len of raft.snapshot[%d]", rf.currentTerm, rf.me, len(rf.persister.snapshot), len(rf.SnapshotDate))
}

func (rf *Raft) persistWithSnapshot() []byte {
	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
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
	d := labgob.NewDecoder(r)

	var currentTerm int
	var votedFor int
	var log []LogType
	var lastIncludeIndex int
	var lastIncludeTerm int
	// var duplicateMap map[int64]duplicateType
	d.Decode(&currentTerm)
	d.Decode(&votedFor)
	d.Decode(&log)
	d.Decode(&lastIncludeIndex)
	d.Decode(&lastIncludeTerm)
	// d.Decode(&duplicateMap)
	rf.currentTerm = currentTerm
	rf.votedFor = votedFor
	rf.log = append(rf.log, log...)
	rf.lastIncludeIndex = lastIncludeIndex
	rf.lastIncludeTerm = lastIncludeTerm
	// rf.duplicateMap = duplicateMap
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

func (rf *Raft) CopyEntries(args *AppendEntriesArgs) {
	logchange := false
	for i := 0; i < len(args.Entries); i++ {
		rfIndex := i + args.PrevLogIndex + 1
		logPos := rf.index2LogPos(rfIndex)
		if rfIndex > rf.lastIndex() { //超出原本log长度了
			rf.log = append(rf.log, args.Entries[i:]...)
			rf.persist()
			logchange = true
			break
		} else if rf.log[logPos].Term != args.Entries[i].Term { //有脏东西
			rf.log = rf.log[:logPos] //删除脏数据
			//一口气复制完
			rf.log = append(rf.log, args.Entries[i:]...)
			rf.persist()
			logchange = true
			break
		}
	}
	//用于debug
	if logchange {
		DPrintf("💖Rev Term[%d] [%d] Copy: Len -> [%d] ", rf.currentTerm, rf.me, len(rf.log))

		DPrintf("Term[%d] [%d] after copy:", rf.currentTerm, rf.me)
		i := len(rf.log) - 10
		if i < 0 {
			i = 0
		}
		for ; i < len(rf.log); i++ {
			DPrintf("Term[%d] [%d] index[%d] log[%v]", rf.currentTerm, rf.me, i+rf.lastIncludeIndex+1, rf.log[i].Value)
		}

	}
	var min = -1
	if args.LeaderCommit > rf.lastIndex() {
		min = rf.lastIndex()
	} else {
		min = args.LeaderCommit
	}
	if rf.commitIndex < min {
		DPrintf("COMIT Term[%d] [%d] CommitIndex: [%d] -> [%d]", rf.currentTerm, rf.me, rf.commitIndex, min)
		rf.commitIndex = min
	}

}
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {

	rf.mu.Lock()
	defer rf.mu.Unlock()
	defer DPrintf("🎫Rec Term[%d] [%d]reply [%d]Vote finish", rf.currentTerm, rf.me, args.CandidateId)
	reply.VoteGranted = false
	reply.Term = rf.currentTerm

	if args.Term < rf.currentTerm {
		return
	}
	if rf.currentTerm < args.Term {
		rf.currentTerm = args.Term
		rf.state = follower
		rf.votedFor = -1
		rf.leaderId = -1
		rf.persist()
	}
	if rf.votedFor == -1 || rf.votedFor == args.CandidateId {
		lastLogTerm := rf.lastTerm()
		if args.LastLogTerm > lastLogTerm || (args.LastLogTerm == lastLogTerm && args.LastLogIndex >= rf.lastIndex()) {
			rf.votedFor = args.CandidateId
			reply.VoteGranted = true
			rf.lastHearBeatTime = time.Now()
			DPrintf("🎫Rec Term[%d] [%d] -> [%d]", rf.currentTerm, rf.me, args.CandidateId)
		}
	}
	rf.persist()
}
func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {

	//需要补充判断条件
	rf.mu.Lock()
	defer rf.mu.Unlock()
	reply.Success = false
	reply.Term = rf.currentTerm
	reply.ConflictIndex = -1
	reply.ConflictTerm = -1

	if rf.currentTerm > args.Term {
		DPrintf("💔Rec Term[%d] [%d] Reject Leader[%d]Term[%d][too OLE]", rf.currentTerm, rf.me, args.LeaderId, args.Term)
		return
	}
	// rf.tryChangeToFollower(args.Term)
	if rf.currentTerm < args.Term {
		rf.currentTerm = args.Term
		rf.state = follower
		rf.votedFor = -1
		rf.persist()
	}
	if rf.currentTerm == args.Term && atomic.LoadInt32(&rf.state) == candidate {
		rf.state = follower
		rf.votedFor = -1
		rf.persist()
	}
	rf.lastHearBeatTime = time.Now()
	rf.leaderId = args.LeaderId
	DPrintf("💖Rec Term[%d] [%d] Receive: LeaderId[%d]Term[%d] PreLogIndex[%d] PrevLogTerm[%d] LeaderCommit[%d] Entries[%v] len[%d]", rf.currentTerm, rf.me, args.LeaderId, args.Term, args.PrevLogIndex, args.PrevLogTerm, args.LeaderCommit, args.Entries, len(args.Entries))
	//新判断
	if args.PrevLogIndex < rf.lastIncludeIndex { // index在快照范围内，那么
		reply.ConflictIndex = 0
		DPrintf("💔Rec Term[%d] [%d] Reject for args.PrevLogIndex[%d] < rf.lastIncludeIndex[%d]", rf.currentTerm, rf.me, args.PrevLogIndex, rf.lastIncludeIndex)
		return
	} else if args.PrevLogIndex == rf.lastIncludeIndex {
		if args.PrevLogTerm != rf.lastIncludeTerm {
			reply.ConflictIndex = 0
			DPrintf("💔Rec Term[%d] [%d] Reject for args.PrevLogTermk[%d] != rf.lastIncludeTerm[%d]", rf.currentTerm, rf.me, args.PrevLogIndex, rf.lastIncludeIndex)
			return
		}
	} else { //index在快照范围外，那么正常走日志覆盖逻辑
		if rf.lastIndex() < args.PrevLogIndex {
			reply.ConflictIndex = rf.lastIndex()
			DPrintf("💔Rec Term[%d] [%d] Reject:PreLogIndex[%d] Out of Len ->[%d]", rf.currentTerm, rf.me, args.PrevLogIndex, rf.lastIndex())
			return
		}

		if args.PrevLogIndex >= 0 && rf.log[rf.index2LogPos(args.PrevLogIndex)].Term != args.PrevLogTerm {
			reply.ConflictTerm = rf.log[rf.index2LogPos(args.PrevLogIndex)].Term
			for index := rf.lastIncludeIndex + 1; index <= args.PrevLogIndex; index++ { // 找到冲突term的首次出现位置，最差就是PrevLogIndex
				if rf.log[rf.index2LogPos(args.PrevLogIndex)].Term == reply.ConflictTerm {
					reply.ConflictIndex = index
					break
				}
			}
			DPrintf("💔Rev Term[%d] [%d] Reject :PreLogTerm Not Match [%d] != [%d]", rf.currentTerm, rf.me, rf.log[rf.index2LogPos(args.PrevLogIndex)].Term, args.PrevLogTerm)
			return
		}
	}
	//保存日志
	rf.CopyEntries(args)
	reply.Success = true
	rf.persist()
}

func (rf *Raft) InstallSnapshot(args *SnapshotInstallArgs, reply *SnapshotInstallreplys) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	DPrintf("SNAPS Term[%d] [%d] Receiv📷 from[%d] lastIncludeIndex[%d] lastIncludeTerm[%d]", rf.currentTerm, rf.me, args.LeaderId, args.LastIncludeIndex, args.LastIncludeTerm)

	reply.Term = rf.currentTerm

	if args.Term < rf.currentTerm {
		DPrintf("SNAPS Term[%d] [%d] reject📷 for it's Term[%d] [too old]", rf.currentTerm, rf.me, args.Term)
		return
	}

	if args.Term > rf.currentTerm {
		rf.currentTerm = args.Term
		rf.state = follower
		rf.votedFor = -1
		rf.persist()
	}

	rf.leaderId = args.LeaderId
	rf.lastHearBeatTime = time.Now()

	if args.LastIncludeIndex <= rf.lastIncludeIndex {
		return
	} else {
		if args.LastIncludeIndex < rf.lastIndex() {
			if rf.log[rf.index2LogPos(args.LastIncludeIndex)].Term != args.LastIncludeTerm {
				rf.log = make([]LogType, 0)
			} else {
				leftLog := make([]LogType, rf.lastIndex()-args.LastIncludeIndex)
				copy(leftLog, rf.log[rf.index2LogPos(args.LastIncludeIndex+1):])
				rf.log = leftLog
				// rf.log = rf.log[rf.index2LogPos(args.LastIncludeIndex+1):]
			}
		} else {
			rf.log = make([]LogType, 0)
		}
	}
	DPrintf("SNAPS Term[%d] [%d] Accept📷 Now it's lastIncludeIndex [%d] -> [%d] lastIncludeTerm [%d] -> [%d]", rf.currentTerm, rf.me, rf.lastIncludeIndex, args.LastIncludeIndex, rf.lastIncludeTerm, args.LastIncludeTerm)
	DPrintf("snaps Term[%d] [%d] after snapshot log:", rf.currentTerm, rf.me)

	rf.lastIncludeIndex = args.LastIncludeIndex
	rf.lastIncludeTerm = args.LastIncludeTerm

	i := len(rf.log) - 10
	if i < 0 {
		i = 0
	}
	for ; i < len(rf.log); i++ {
		DPrintf("Term[%d] [%d] index[%d] value[%v]", rf.currentTerm, rf.me, i+rf.lastIncludeIndex+1, rf.log[i])
	}

	DPrintf("📷Cmi Term[%d] [%d] 📦Save snapshot to application[%d] (Receive from leader)", rf.currentTerm, rf.me, len(rf.persister.snapshot))
	//snapshot提交给应用层
	applyMsg := ApplyMsg{
		SnapshotValid: true,
		Snapshot:      args.Data,
		SnapshotIndex: rf.lastIncludeIndex + 1, //记得这里有个坑
		SnapshotTerm:  rf.lastIncludeTerm,
	}
	//快照提交给了application
	rf.lastApplied = rf.lastIncludeIndex
	DPrintf("📷Cmi Term[%d] [%d] Ready to commit snapshot snapshotIndex[%d] snapshotTerm[%d]", rf.currentTerm, rf.me, rf.lastIncludeIndex, rf.lastIncludeTerm)
	rf.mu.Unlock()
	rf.applyChTerm <- applyMsg
	rf.mu.Lock()
	//持久化快照
	rf.SnapshotDate = args.Data
	rf.persister.Save(rf.persistWithSnapshot(), args.Data)
	DPrintf("📷Cmi Term[%d] [%d] Done Success to comit snapshot snapshotIndex[%d] snapshotTerm[%d]", rf.currentTerm, rf.me, rf.lastIncludeIndex, rf.lastIncludeTerm)
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
	if len(rf.persister.snapshot) < 1 {
		DPrintf("📷Cmi Term[%d] [%d] Snapshotlen[%d] No need to commit snapshotIndex[%d] snapshotTerm[%d] ", rf.currentTerm, rf.me, len(rf.persister.snapshot), rf.lastIncludeIndex, rf.lastIncludeTerm)
		return
	}
	rf.SnapshotDate = rf.persister.ReadSnapshot()
	rf.lastApplied = rf.lastIncludeIndex
	DPrintf("📷Cmi Term[%d] [%d] Ready to commit snapshot snapshotIndex[%d] snapshotTerm[%d]", rf.currentTerm, rf.me, rf.lastIncludeIndex, rf.lastIncludeTerm)
	rf.applyChTerm <- *applyMsg
	DPrintf("📷Cmi Term[%d] [%d] Done Success to comit snapshot snapshotIndex[%d] snapshotTerm[%d]", rf.currentTerm, rf.me, rf.lastIncludeIndex, rf.lastIncludeTerm)
}

func (rf *Raft) sendInstallSnapshot(server int, args *SnapshotInstallArgs, reply *SnapshotInstallreplys) bool {
	ok := rf.peers[server].Call("Raft.InstallSnapshot", args, reply)
	return ok
}

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// Your code here (3D).
	DPrintf("SNAPS Term[%d] [%d] 📷Snapshot ask to snap Index[%d] Raft log Len:[%d]", rf.currentTerm, rf.me, index-1, len(rf.log))
	// DPrintf("SNAPS Term[%d] [%d] Wait for the lock🤨", rf.currentTerm, rf.me)
	rf.mu.Lock()
	// DPrintf("SNAPS Term[%d] [%d] Get the lock🔐", rf.currentTerm, rf.me)
	// defer DPrintf("SNAPS Term[%d] [%d] Unlock the lock🔓", rf.currentTerm, rf.me)
	defer rf.mu.Unlock()

	index -= 1
	if index <= rf.lastIncludeIndex {
		return
	}
	compactLoglen := index - rf.lastIncludeIndex
	DPrintf("SNAPS Term[%d] [%d] After📷,lastIncludeIndex[%d]->[%d] lastIncludeTerm[%d]->[%d] len of Log->[%d]", rf.currentTerm, rf.me, rf.lastIncludeIndex, index, rf.lastIncludeTerm, rf.log[rf.index2LogPos(index)].Term, len(rf.log)-compactLoglen)

	rf.lastIncludeTerm = rf.log[rf.index2LogPos(index)].Term
	rf.lastIncludeIndex = index

	//压缩日志
	afterLog := make([]LogType, len(rf.log)-compactLoglen)
	copy(afterLog, rf.log[compactLoglen:])
	rf.log = afterLog
	//把snapshot和raftstate持久化
	rf.SnapshotDate = snapshot
	rf.persister.Save(rf.persistWithSnapshot(), snapshot)
	DPrintf("📷Cmi Term[%d] [%d] 📦Save snapshot to application[%d] (Receive from up Application)", rf.currentTerm, rf.me, len(rf.persister.snapshot))
}

const (
	AEtry = iota
	AElostRPC
	AERejectRPC
)

func (rf *Raft) updateCommitIndex() {
	//从matchIndex寻找一个大多数服务器认同的N
	for !rf.killed() {
		time.Sleep(time.Millisecond * time.Duration(10))
		_, isleader := rf.GetState()
		if isleader {
			rf.mu.Lock()
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
			if N > rf.commitIndex && (N <= rf.lastIncludeIndex || rf.currentTerm == rf.log[rf.index2LogPos(N)].Term) {
				// DPrintf("COMIT Term[%d] [%d] It's matchIndex = %v", rf.currentTerm, rf.me, matchIndex)
				DPrintf("COMIT Term[%d] [%d] commitIndex [%d] -> [%d] (leader action)", rf.currentTerm, rf.me, rf.commitIndex, N)
				rf.IisBackIndex = N
				rf.commitIndex = N
			}
			rf.mu.Unlock()
		}

	}
}

// 修改rf.lastApplied
func (rf *Raft) undateLastApplied() {
	var nomore = false
	for !rf.killed() {
		if nomore {
			time.Sleep(time.Millisecond * time.Duration(10))
		}

		func() {
			// rf.snapshotXapplych.Lock()
			// defer rf.snapshotXapplych.Unlock()
			// DPrintf("APPLY Term[%d] [%d] Wait for the lock🔐", rf.currentTerm, rf.me)
			rf.mu.Lock()
			// DPrintf("APPLY Term[%d] [%d] Hode the lock🔐", rf.currentTerm, rf.me)
			nomore = true
			if rf.lastApplied < rf.commitIndex {
				rf.lastApplied += 1
				index := rf.index2LogPos(rf.lastApplied)
				if index <= -1 || index >= len(rf.log) {
					rf.lastApplied = rf.lastIncludeIndex
					DPrintf("ERROR? 👿 [%d]Ready to apply index[%d] But index out of Len of log, lastApplied[%d] commitIndex[%d] lastIncludeIndex[%d] logLen:%d", rf.me, index, rf.lastApplied, rf.commitIndex, rf.lastIncludeIndex, len(rf.log))
					rf.mu.Unlock()
					return
				}
				DPrintf("APPLY Term[%d] [%d] -> LOG [%d] value:[%d]", rf.currentTerm, rf.me, rf.lastApplied, rf.log[index].Value)

				ApplyMsg := ApplyMsg{
					CommandValid: true,
					Command:      rf.log[index].Value,
					CommandIndex: rf.lastApplied + 1,
				}
				DPrintf("APPLY Term[%d] [%d] Unlock the lock🔐 For Start applyerCh <- len[%d]", rf.currentTerm, rf.me, len(rf.applyChTerm))
				rf.applyChTerm <- ApplyMsg
				if rf.IisBackIndex == rf.lastApplied {
					DPrintf("Term [%d] [%d] iisback = true iisbackIndex =[%d]", rf.currentTerm, rf.me, rf.IisBackIndex)
					rf.IisBack = true
				}
				// DPrintf("APPLY Term[%d] [%d] lock the lock🔐 For Finish applyerCh <-", rf.currentTerm, rf.me)
				nomore = false
				DPrintf("APPLY Term[%d] [%d] AppliedIndex [%d] CommitIndex [%d]", rf.currentTerm, rf.me, rf.lastApplied, rf.commitIndex)
			}
			rf.mu.Unlock()
			// DPrintf("APPLY Term[%d] [%d] Open the lock🔓", rf.currentTerm, rf.me)
		}()

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

func (rf *Raft) Start(command interface{}) (int, int, bool) {
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
	rf.log = append(rf.log, LogType{
		Term:  rf.currentTerm,
		Value: command,
	})
	rf.persist()

	DPrintf("CLIENT📨 Term[%d] [%d] Receive [%v] logIndex[%d](leader action)\n", rf.currentTerm, rf.me, command, index-1)
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

func (rf *Raft) sendRequestVote2(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

func (rf *Raft) electionLoop() {
	for !rf.killed() {
		time.Sleep(time.Millisecond * 10)
		func() {
			rf.mu.Lock()
			defer rf.mu.Unlock()

			timeCount := time.Since(rf.lastHearBeatTime).Milliseconds()
			ms := TICKMIN + rand.Int63()%TICKRANDOM
			if rf.state == follower {
				if timeCount >= ms {
					DPrintf("❗Term[%d] [%d] Follower -> Candidate", rf.currentTerm, rf.me)
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
				args := RequestVoteArgs{
					Term:         rf.currentTerm,
					CandidateId:  rf.me,
					LastLogIndex: rf.lastIndex(),
					LastLogTerm:  rf.lastTerm(),
				}
				rf.mu.Unlock()

				type VoteResult struct {
					raftId int
					resp   *RequestVoteReply
				}
				voteCount := 1
				finishCount := 1
				VoteResultChan := make(chan *VoteResult, len(rf.peers))
				for peerId := 0; peerId < len(rf.peers) && !rf.killed(); peerId++ {
					go func(server int) {
						if server == rf.me {
							return
						}
						resp := RequestVoteReply{}
						if ok := rf.sendRequestVote2(server, &args, &resp); ok {

							if resp.VoteGranted {
								DPrintf("🎫Get Term[%d] [%d]Candidate 🥰receive a voteRPC reply from [%d] ,voteGranted Yes", rf.currentTerm, rf.me, server)
							} else {
								DPrintf("🎫Get Term[%d] [%d]Candidate 🥰receive a voteRPC reply from [%d] ,voteGranted No", rf.currentTerm, rf.me, server)
							}
							VoteResultChan <- &VoteResult{raftId: server, resp: &resp}

						} else {
							DPrintf("🎫Get Term[%d] [%d]Candidate 🥲Do not get voteRPC reply from [%d] ,voteGranted Nil", rf.currentTerm, rf.me, server)
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
							if VoteResult.resp.Term > maxTerm {
								maxTerm = VoteResult.resp.Term
							}
						}

						if finishCount == len(rf.peers) || voteCount > len(rf.peers)/2 {
							goto VOTE_END
						}
					case <-time.After(time.Duration(TICKMIN+rand.Int63()%TICKRANDOM) * time.Millisecond):
						DPrintf("🎫Get Term[%d] [%d]Candidate Fail🥲 election time out", rf.currentTerm, rf.me)
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
					DPrintf("❗ Term[%d] [%d]candidate -> leader", rf.currentTerm, rf.me)
					rf.lastSendHeartbeatTime = time.Now().Add(-time.Millisecond * 2 * HeartBeatInterval)
					return
				}
				DPrintf("🎫Rec Term[%d] [%d]candidate Fail to get majority Vote", rf.currentTerm, rf.me)
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
	args := &AppendEntriesArgs{}
	reply := &AppendEntriesReply{}
	*args = AppendEntriesArgs{
		Term:         rf.currentTerm,
		LeaderId:     rf.me,
		PrevLogIndex: rf.nextIndex[server] - 1,
		PrevLogTerm:  -1,
		Entries:      make([]LogType, 0),
		LeaderCommit: rf.commitIndex,
	}
	if args.PrevLogIndex == rf.lastIncludeIndex {
		args.PrevLogTerm = rf.lastIncludeTerm
	}

	if rf.index2LogPos(args.PrevLogIndex) >= 0 && rf.index2LogPos(args.PrevLogIndex) < len(rf.log) { //有PrevIndex
		args.PrevLogTerm = rf.log[rf.index2LogPos(args.PrevLogIndex)].Term
	}
	if rf.index2LogPos(rf.nextIndex[server]) >= 0 && rf.index2LogPos(rf.nextIndex[server]) < len(rf.log) { //有nextIndex
		args.Entries = append(args.Entries, rf.log[rf.index2LogPos(rf.nextIndex[server]):]...)
	}
	rf.mu.Unlock()

	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)

	rf.mu.Lock()
	defer rf.mu.Unlock()
	if ok {
		if args.Term != rf.currentTerm {
			DPrintf("💔Rec Term[%d] [%d] Receive Send.Term[%d][too OLD]", rf.currentTerm, rf.me, args.Term)
			if applychreply != nil {
				*applychreply <- AEresult_Ignore
			}
			return
		}
		if rf.currentTerm < reply.Term {
			rf.votedFor = -1
			rf.state = follower
			rf.currentTerm = reply.Term
			rf.leaderId = -1
			rf.persist()
			DPrintf("💔Rec Term[%d] [%d] Receive Discover newer Term[%d]", rf.currentTerm, rf.me, reply.Term)
			if applychreply != nil {
				*applychreply <- AEresult_StopSending
			}
			return
		}
		if reply.Success {
			rf.nextIndex[server] = args.PrevLogIndex + len(args.Entries) + 1
			rf.matchIndex[server] = rf.nextIndex[server] - 1 //做闭右开，因此curLatestIndex指向的是最后一个发送的log的下一位可能为空
			if applychreply != nil {
				*applychreply <- AEresult_Accept
			}
			return
		} else {

			if reply.ConflictTerm != -1 {
				searchIndex := -1
				for i := args.PrevLogIndex; i > rf.lastIncludeIndex; i-- {
					if rf.log[rf.index2LogPos(i)].Term == reply.ConflictTerm {
						searchIndex = i
					}
				}
				if searchIndex != -1 {
					rf.nextIndex[server] = searchIndex + 1
				} else {
					rf.nextIndex[server] = reply.ConflictIndex
				}
			} else {
				rf.nextIndex[server] = reply.ConflictIndex + 1
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
		time.Sleep(10 * time.Millisecond)
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
	// DPrintf("SNAPS Term[%d] [%d] goSend📷Wait for a lock🤨 to [%d],", rf.currentTerm, rf.me, server)
	rf.mu.Lock()
	args := &SnapshotInstallArgs{}
	args.Term = rf.currentTerm
	args.LeaderId = rf.me
	args.LastIncludeIndex = rf.lastIncludeIndex
	args.LastIncludeTerm = rf.lastIncludeTerm
	args.Data = rf.SnapshotDate
	DPrintf("SNAPS Term[%d] [%d] goSend📷 to [%d] args.LastIncludeIndex[%d],args.LastIncludeTerm[%d],len of snapshot[%d],", rf.currentTerm, rf.me, server, args.LastIncludeIndex, args.LastIncludeTerm, len(args.Data))
	rf.mu.Unlock()
	go func(args *SnapshotInstallArgs) {
		reply := &SnapshotInstallreplys{}
		if rf.sendInstallSnapshot(server, args, reply) {
			rf.mu.Lock()
			defer rf.mu.Unlock()
			if rf.currentTerm != args.Term {
				return
			}
			if reply.Term > rf.currentTerm {
				rf.state = follower
				rf.leaderId = -1
				rf.currentTerm = reply.Term
				rf.votedFor = -1
				rf.persist()
				return
			}
			DPrintf("SNAPS Term[%d] [%d] leader success to Send a 📷 to [%d] nextIndex for it [%d] -> [%d] matchIndex [%d] -> [%d]", rf.currentTerm, rf.me, server, rf.nextIndex[server], rf.lastIndex()+1, rf.matchIndex[server], args.LastIncludeIndex)
			rf.nextIndex[server] = rf.lastIndex() + 1
			rf.matchIndex[server] = args.LastIncludeIndex
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
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me

	//state
	rf.state = follower
	rf.leaderId = -1
	//persister
	rf.currentTerm = 0
	rf.votedFor = -1
	rf.log = make([]LogType, 0, LOGINITCAPCITY)
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
	DPrintf("RESTA Term[%d] [%d] Restart😎", rf.currentTerm, rf.me)
	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())
	// 向application层安装快照
	rf.installSnapshotToApplication()
	// start ticker goroutine to start elections
	// go rf.mainLoop2()
	go rf.Applyer()
	go rf.electionLoop()
	go rf.appendEntriesLoop()
	go rf.undateLastApplied()
	go rf.updateCommitIndex()
	return rf
}
