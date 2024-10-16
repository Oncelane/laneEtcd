package client

import (
	"encoding/json"
	"time"

	"github.com/Oncelane/laneEtcd/src/kvraft"
	"github.com/Oncelane/laneEtcd/src/pkg/laneLog"
)

type Node struct {
	Name     string
	AppId    string
	Port     string
	IPs      []string
	Location string
	Connect  int32
	Weight   int32
	Env      string
	MetaDate map[string]string //"color" "version"
}

func (n *Node) Marshal() []byte {
	data, err := json.Marshal(n)
	if err != nil {
		laneLog.Logger.Fatalln(err)
	}
	return data
}

func (n *Node) Unmarshal(data []byte) {
	err := json.Unmarshal(data, n)
	if err != nil {
		laneLog.Logger.Fatalln(err)
	}
}

func (n *Node) Key() string {
	return n.Name
}

func (n *Node) SetNode(ck *Clerk, TTL time.Duration) error {
	err := ck.Put(n.Key(), n.Marshal(), TTL)
	if err != nil {
		laneLog.Logger.Fatal(err)
	}
	return err
}

func (n *Node) SetNode_Watch(ck *Clerk) (cancle func()) {
	return ck.WatchDog(n.Key(), n.Marshal())
}

func GetNode(ck *Clerk, name string) ([]*Node, error) {
	datas, err := ck.GetWithPrefix(name)
	if err != nil {
		if err == kvraft.ErrNil {
			return nil, nil
		}
		laneLog.Logger.Fatalln(err)
		return nil, err
	}
	// laneLog.Logger.Debugln("raw data:", datas)
	nodes := make([]*Node, len(datas))
	for i := range datas {
		n := &Node{}
		n.Unmarshal([]byte(datas[i]))
		nodes[i] = n
	}
	return nodes, nil
}
