package main

import (
	"flag"
	"runtime"

	"github.com/Oncelane/laneEtcd/src/kvraft"
	"github.com/Oncelane/laneEtcd/src/pkg/laneConfig"
	"github.com/Oncelane/laneEtcd/src/pkg/laneLog"
	"github.com/Oncelane/laneEtcd/src/raft"
)

var (
	ConfigPath = flag.String("c", "config.yml", "path fo config.yml folder")
)

func main() {
	runtime.GOMAXPROCS(1)
	flag.Parse()
	conf := laneConfig.Kvserver{}
	laneConfig.Init(*ConfigPath, &conf)

	// conf.Clients[conf.Me].Addr+conf.Clients[conf.Me].Addr

	laneLog.InitLogger("kvserver", true, true, true)
	_ = kvraft.StartKVServer(conf, conf.Rafts.Me, raft.MakePersister("/raftstate.dat", "/snapshot.dat", conf.DataBasePath), conf.Maxraftstate)
	select {}
}
