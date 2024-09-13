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
}
