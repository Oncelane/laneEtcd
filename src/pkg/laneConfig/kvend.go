package laneConfig

type Kvserver struct {
	Addr  string
	Port  string
	Rafts RaftEnds
}

func (c *Kvserver) Default() {
	*c = DefaultKVEnd()
}

func DefaultKVEnd() Kvserver {
	return Kvserver{
		Addr:  "127.0.0.1",
		Port:  ":51242",
		Rafts: DefaultRaftEnds(),
	}
}
