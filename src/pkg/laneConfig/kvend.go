package laneConfig

type Kvserver struct {
	Addr         string
	Port         string
	Rafts        RaftEnds
	DataBasePath string
	Maxraftstate int
}

func (c *Kvserver) Default() {
	*c = DefaultKVServer()
}

func DefaultKVServer() Kvserver {
	return Kvserver{
		Addr:         "127.0.0.1",
		Port:         ":51242",
		Rafts:        DefaultRaftEnds(),
		DataBasePath: "data",
		Maxraftstate: 100000,
	}
}
