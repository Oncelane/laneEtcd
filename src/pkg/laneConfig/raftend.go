package laneConfig

type RaftEnds struct {
	Me        int
	Endpoints []RaftEnd
}

type RaftEnd struct {
	Addr string
	Port string
}

func (c *RaftEnds) Default() {
	*c = DefaultRaftEnds()
}

func DefaultRaftEnds() RaftEnds {
	return RaftEnds{
		Me: 0,
		Endpoints: []RaftEnd{
			{
				Addr: "127.0.0.1",
				Port: ":32300",
			},
			{
				Addr: "127.0.0.1",
				Port: ":32301",
			},
			{
				Addr: "127.0.0.1",
				Port: ":32302",
			},
		},
	}
}
