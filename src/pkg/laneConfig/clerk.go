package laneConfig

type Clerk struct {
	EtcdAddrs []string
}

func (c *Clerk) Default() {
	*c = DefaultClerk()
}

func DefaultClerk() Clerk {
	return Clerk{
		EtcdAddrs: []string{
			"127.0.0.1:32300", "127.0.0.1:32301", "127.0.0.1:32302",
		},
	}
}
