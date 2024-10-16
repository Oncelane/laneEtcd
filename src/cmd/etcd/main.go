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
	if len(conf.Rafts.Endpoints)%2 == 0 {
		laneLog.Logger.Fatalln("the number of nodes is not odd")
	}
	if len(conf.Rafts.Endpoints) < 3 {
		laneLog.Logger.Fatalln("the number of nodes is less than 3")
	}
	// conf.Endpoints[conf.Me].Addr+conf.Endpoints[conf.Me].Addr

	laneLog.InitLogger("kvserver", true, false, false)
	_ = kvraft.StartKVServer(conf, conf.Rafts.Me, raft.MakePersister("/raftstate.dat", "/snapshot.dat", conf.DataBasePath), conf.Maxraftstate)
	select {}
}
