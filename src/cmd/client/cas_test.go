package client_test

import (
	"testing"

	"github.com/Oncelane/laneEtcd/src/pkg/laneLog"
)

func TestCAS(t *testing.T) {
	key := "cas"
	ok, err := ck.CAS(key, "", "SetNX", 0)
	if err != nil {
		laneLog.Logger.Infoln(err)
	}
	if ok {
		laneLog.Logger.Infoln("success set cas")
	} else {
		laneLog.Logger.Infoln("faild set cas")
	}

	rt, err := ck.Get(key)
	if err != nil {
		laneLog.Logger.Infoln(err)
	}
	laneLog.Logger.Infof("get key[%s] value[%s]", key, rt)

	laneLog.Logger.Infoln("reset SetNX")
	ok, err = ck.CAS(key, "", "SetNX", 0)
	if err != nil {
		laneLog.Logger.Infoln(err)
	}
	if ok {
		laneLog.Logger.Infoln("success set cas")
	} else {
		laneLog.Logger.Infoln("faild set cas")
	}

	rt, err = ck.Get(key)
	if err != nil {
		laneLog.Logger.Infoln(err)
	}
	laneLog.Logger.Infof("get key[%s] value[%s]", key, rt)

	laneLog.Logger.Infoln("reset SetNX")
	ok, err = ck.CAS(key, "SetNX", "SetEX", 0)
	if err != nil {
		laneLog.Logger.Infoln(err)
	}
	if ok {
		laneLog.Logger.Infoln("success set cas")
	} else {
		laneLog.Logger.Infoln("faild set cas")
	}

	rt, err = ck.Get(key)
	if err != nil {
		laneLog.Logger.Infoln(err)
	}
	laneLog.Logger.Infof("get key[%s] value[%s]", key, rt)

	err = ck.Delete(key)
	if err != nil {
		laneLog.Logger.Infoln(err)
	}
}
