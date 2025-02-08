package client_test

import (
	"testing"
	"time"

	"github.com/Oncelane/laneEtcd/src/pkg/laneLog"
)

func TestWatchDog(t *testing.T) {

	cancel := ck.WatchDog("watchdog", []byte("exist"))

	go func() {
		for {
			ret, err := ck.Get("watchdog")
			if err != nil {
				laneLog.Logger.Fatalln(err)
			}
			laneLog.Logger.Infoln(string(ret))
			time.Sleep(time.Millisecond * 500)
		}
	}()
	time.Sleep(time.Second * 2)
	laneLog.Logger.Infoln("after cancel")
	cancel()
	time.Sleep(time.Second * 5)
}
