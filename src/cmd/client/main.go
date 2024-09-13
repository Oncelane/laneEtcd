package main

import (
	"flag"
	kvraft "laneEtcd/src/kvServer"
	"laneEtcd/src/pkg/laneConfig"
)

var (
	ConfigPath = flag.String("c", "config.yml", "path fo config.yml folder")
)

func main() {

	flag.Parse()
	conf := laneConfig.Clerk{}
	laneConfig.Init(*ConfigPath, &conf)
	kvraft.NewClerk(conf)
}
