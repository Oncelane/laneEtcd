package client_test

import (
	"encoding/json"
	"math/rand"
	"strconv"
	"testing"
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

func (n *Node) SetNode(ck *kvraft.Clerk, TTL time.Duration) {
	err := ck.Put(n.Key(), string(n.Marshal()), TTL)
	if err != nil {
		laneLog.Logger.Fatal(err)
	}
}

func GetNode(ck *kvraft.Clerk, name string) ([]*Node, error) {
	datas, err := ck.GetWithPrefix(name)
	if err != nil {
		laneLog.Logger.Fatalln(err)
		return nil, err
	}
	// laneLog.Logger.Debugln("raw data:", datas)
	nodes := make([]*Node, len(datas))
	for i := range datas {
		// laneLog.Logger.Debugln(datas[i])
		n := &Node{}
		n.Unmarshal([]byte(datas[i]))
		nodes[i] = n
	}
	return nodes, nil
}

func TestNamingMarshal(t *testing.T) {
	n := Node{
		Name:     "comet",
		AppId:    "v1.0",
		Port:     ":8020",
		Location: "sz",
		Env:      "produce",
		MetaDate: map[string]string{"color": "red"},
	}
	data := n.Marshal()
	nn := Node{}
	nn.Unmarshal(data)
	laneLog.Logger.Infoln("unmashal:", nn)
}

func TestNaming(t *testing.T) {
	n := Node{
		Name:     "comet",
		AppId:    "v1.0",
		Port:     ":8020",
		Location: "sz",
		Env:      "produce",
		MetaDate: map[string]string{"color": "red"},
	}
	for i := range 4 {
		n.Name = "comet" + strconv.Itoa(i)
		n.IPs = []string{"localhost" + strconv.Itoa(i)}
		n.Connect = int32(rand.Int() % 10000)
		timeout := time.Millisecond * 800
		n.SetNode(ck, timeout)
	}
	laneLog.Logger.Infoln("set node success")

	nodes, err := GetNode(ck, "comet")
	if err != nil {
		laneLog.Logger.Fatalln(err)
	}
	for _, n := range nodes {
		laneLog.Logger.Infof("get nodes:%+v", n)
	}
	time.Sleep(time.Second)
	nodes, err = GetNode(ck, "comet")
	if err != nil {
		laneLog.Logger.Fatalln(err)
	}
	if len(nodes) == 0 {
		laneLog.Logger.Infoln("no node")
	}
	for _, n := range nodes {
		laneLog.Logger.Infof("get nodes:%+v", n)
	}
}
