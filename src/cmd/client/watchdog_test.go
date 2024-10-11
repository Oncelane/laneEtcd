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
			laneLog.Logger.Infoln(ck.Get("watchdog"))
			time.Sleep(time.Millisecond * 500)
		}
	}()
	time.Sleep(time.Second * 5)
	cancel()
	time.Sleep(time.Second * 5)
}
