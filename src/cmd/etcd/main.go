package main

import (
	"flag"

	kvraft "github.com/Oncelane/laneEtcd/src/kvServer"
	"github.com/Oncelane/laneEtcd/src/pkg/laneConfig"
	"github.com/Oncelane/laneEtcd/src/pkg/laneLog"
	"github.com/Oncelane/laneEtcd/src/raft"
)

var (
	ConfigPath = flag.String("c", "config.yml", "path fo config.yml folder")
)

func main() {

	flag.Parse()
	conf := laneConfig.Kvserver{}
	laneConfig.Init(*ConfigPath, &conf)

	// conf.Clients[conf.Me].Addr+conf.Clients[conf.Me].Addr

	laneLog.InitLogger("kvserver", true)
	_ = kvraft.StartKVServer(conf, conf.Rafts.Me, raft.MakePersister(), 100000)
	select {}
}
