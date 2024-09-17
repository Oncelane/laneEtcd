package main

import (
	"flag"

	kvraft "github.com/Oncelane/laneEtcd/src/kvServer"
	"github.com/Oncelane/laneEtcd/src/pkg/laneConfig"
	"github.com/Oncelane/laneEtcd/src/pkg/laneLog"
)

var (
	ConfigPath = flag.String("c", "config.yml", "path fo config.yml folder")
)

func main() {

	flag.Parse()
	conf := laneConfig.Clerk{}
	laneConfig.Init(*ConfigPath, &conf)
	laneLog.Logger.Debugln("check conf", conf)
	ck := kvraft.MakeClerk(conf)
	ck.Put("logic", "testLogicAddr")
	laneLog.Logger.Infoln("put success")
	r, err := ck.Get("logic")
	laneLog.Logger.Infoln("get success r= ", r, "err=", err)
	r, err = ck.Get("no logic")
	laneLog.Logger.Infoln("get success r= ", r, "err=", err)

	ck.Put("logic:0", "testLogicAddr0 ")
	ck.Put("logic:1", "testLogicAddr1 ")
	ck.Put("logic:2", "testLogicAddr2 ")
	ck.Put("logic:3", "testLogicAddr3 ")
	laneLog.Logger.Infoln("put success")
	laneLog.Logger.Infoln("put success")
	rt, err := ck.GetWithPrefix("logic")
	laneLog.Logger.Infoln("get success r= ", rt, "err=", err)
}
