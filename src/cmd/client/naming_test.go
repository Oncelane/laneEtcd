package client_test

import (
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/Oncelane/laneEtcd/src/client"
	"github.com/Oncelane/laneEtcd/src/pkg/laneLog"
)

func TestNamingMarshal(t *testing.T) {
	n := client.Node{
		Name:     "comet",
		AppId:    "v1.0",
		Port:     ":8020",
		Location: "sz",
		Env:      "produce",
		MetaDate: map[string]string{"color": "red"},
	}
	data := n.Marshal()
	nn := client.Node{}
	nn.Unmarshal(data)
	laneLog.Logger.Infoln("unmashal:", nn)
}

func TestNaming(t *testing.T) {
	n := client.Node{
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
	laneLog.Logger.Infoln("success set node TTL 800ms ")

	// check node
	nodes, err := client.GetNode(ck, "comet")
	if err != nil {
		laneLog.Logger.Fatalln(err)
	}
	for _, n := range nodes {
		laneLog.Logger.Infof("get nodes:%+v", n)
	}

	// after 1 sercond
	time.Sleep(time.Second)
	laneLog.Logger.Infoln("sleep 1000ms")

	// check node
	nodes, err = client.GetNode(ck, "comet")
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

func TestNamingWatch(t *testing.T) {
	n := client.Node{
		Name:     "comet",
		AppId:    "v1.0",
		Port:     ":8020",
		Location: "sz",
		Env:      "produce",
		MetaDate: map[string]string{"color": "red"},
	}
	cancles := make([]func(), 0, 4)
	for i := range 4 {
		n.Name = "comet" + strconv.Itoa(i)
		n.IPs = []string{"localhost" + strconv.Itoa(i)}
		n.Connect = int32(rand.Int() % 10000)
		cancles = append(cancles, n.SetNode_Watch(ck))
	}
	laneLog.Logger.Infoln("success set node WatchDog ")

	// check node
	nodes, err := client.GetNode(ck, "comet")
	if err != nil {
		laneLog.Logger.Fatalln(err)
	}
	for _, n := range nodes {
		laneLog.Logger.Infof("get nodes:%+v", n)
	}

	// simulate server crash or cacle
	laneLog.Logger.Infoln("simulate server crash or cancle ")
	// use cancel func
	for _, f := range cancles {
		f()
	}

	// check node
	laneLog.Logger.Infoln("after cancle-immidiately")
	nodes, err = client.GetNode(ck, "comet")
	if err != nil {
		laneLog.Logger.Fatalln(err)
	}
	if len(nodes) == 0 {
		laneLog.Logger.Infoln("no node")
	}
	for _, n := range nodes {
		laneLog.Logger.Infof("get nodes:%+v", n)
	}

	// check node
	time.Sleep(time.Second * 6)
	laneLog.Logger.Infoln("after cancle 6 second")
	nodes, err = client.GetNode(ck, "comet")
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
