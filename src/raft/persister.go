package raft

//
// support for Raft and kvraft to save persistent
// Raft state (log &c) and k/v server snapshots.
//
// we will use the original persister.go to test your code for grading.
// so, while you can modify this code to help you debug, please
// test with the original before submitting.
//

import (
	"os"
	"sync"

	"github.com/Oncelane/laneEtcd/src/pkg/laneLog"
)

type Persister struct {
	mu            sync.Mutex
	raftstatePath string
	snapshotPath  string
	basePath      string
}

func MakePersister(raftstatePath, snapshotPath, basePath string) *Persister {
	return &Persister{
		raftstatePath: raftstatePath,
		snapshotPath:  snapshotPath,
		basePath:      basePath,
	}
}

func (ps *Persister) Copy() *Persister {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	np := MakePersister(ps.raftstatePath, ps.snapshotPath, ps.basePath)
	return np
}

func (ps *Persister) ReadRaftState() []byte {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	data, err := os.ReadFile(ps.basePath + ps.raftstatePath)

	if err != nil {
		laneLog.Logger.Errorln("read faild", err)
		return nil
	}
	return data
}

func (ps *Persister) RaftStateSize() int {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	info, err := os.Stat(ps.basePath + ps.raftstatePath)
	if err != nil {
		laneLog.Logger.Panicln("read faild", err)
		return 0
	}
	return int(info.Size())
}

func (ps *Persister) Save(raftstate []byte, snapshot []byte) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if _, err := os.Stat(ps.basePath); os.IsNotExist(err) {
		err := os.Mkdir(ps.basePath, os.ModePerm)
		if err != nil {
			laneLog.Logger.Errorf("could not create dir file: %v", err)
		}
	}

	// Create temporary files to ensure atomicity
	raftstateTmpPath := ps.basePath + ps.raftstatePath + ".tmp"
	snapshotTmpPath := ps.basePath + ps.snapshotPath + ".tmp"

	if _, err := os.Stat(raftstateTmpPath); os.IsExist(err) {
		err := os.Remove(raftstateTmpPath)
		if err != nil {
			laneLog.Logger.Errorf("could not remove tmp file: %v", err)
		}
	}
	if _, err := os.Stat(snapshotTmpPath); os.IsExist(err) {
		err := os.Remove(snapshotTmpPath)
		if err != nil {
			laneLog.Logger.Errorf("could not remove tmp file: %v", err)
		}
	}
	if err := os.WriteFile(raftstateTmpPath, raftstate, 0644); err != nil {
		laneLog.Logger.Panicln("write faild", err)
		return
	}
	if err := os.WriteFile(snapshotTmpPath, snapshot, 0644); err != nil {
		laneLog.Logger.Panicln("write faild", err)
		return
	}

	// Rename temp files to final filenames
	if err := os.Rename(raftstateTmpPath, ps.basePath+ps.raftstatePath); err != nil {
		laneLog.Logger.Panicln("write faild", err)
		return
	}
	if err := os.Rename(snapshotTmpPath, ps.basePath+ps.snapshotPath); err != nil {
		laneLog.Logger.Panicln("write faild", err)
		return
	}

}

func (ps *Persister) ReadSnapshot() []byte {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	data, err := os.ReadFile(ps.basePath + ps.snapshotPath)
	if err != nil {
		laneLog.Logger.Errorln("read faild", err)
		return nil
	}
	return data
}

func (ps *Persister) SnapshotSize() int {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	info, err := os.Stat(ps.basePath + ps.snapshotPath)
	if err != nil {
		laneLog.Logger.Errorln("read faild", err)
		return 0
	}
	return int(info.Size())
}
