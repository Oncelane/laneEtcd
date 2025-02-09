package main

import (
	"github.com/Oncelane/laneEtcd/src/gateway"
	"github.com/Oncelane/laneEtcd/src/pkg/laneConfig"
)

var gate *gateway.Gateway

func init() {
	var conf laneConfig.Gateway
	laneConfig.Init("config.yml", &conf)
	gate = gateway.NewGateway(conf)
}
func main() {
	gate.Run()
}
