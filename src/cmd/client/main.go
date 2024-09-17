package main

import (
	"flag"
	kvraft "laneEtcd/src/kvServer"
	"laneEtcd/src/pkg/laneConfig"
	"laneEtcd/src/pkg/laneLog"
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
	laneLog.Logger.Infoln("get success:", ck.Get("logic"))

	ck.Put("logic:0", "testLogicAddr0 ")
	ck.Put("logic:1", "testLogicAddr1 ")
	ck.Put("logic:2", "testLogicAddr2 ")
	ck.Put("logic:3", "testLogicAddr3 ")
	laneLog.Logger.Infoln("put success")
	laneLog.Logger.Infoln("get success:", ck.GetWithPrefix("logic"))
}
