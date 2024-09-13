package laneConfig

type Kvserver struct {
	Addr  string
	Port  string
	Rafts RaftEnds
}

func (c *Kvserver) Default() {
	*c = DefaultKVServer()
}

func DefaultKVServer() Kvserver {
	return Kvserver{
		Addr:  "127.0.0.1",
		Port:  ":51242",
		Rafts: DefaultRaftEnds(),
	}
}
