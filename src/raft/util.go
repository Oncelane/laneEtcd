package raft

import (
	"github.com/Oncelane/laneEtcd/src/pkg/laneLog"
)

// Debugging
const Debug = true

func DPrintf(format string, a ...interface{}) {
	if Debug {
		laneLog.Logger.Infof(format, a...)
	}
}
