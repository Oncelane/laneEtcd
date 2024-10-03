package client_test

import (
	"sync"
	"testing"
	"time"

	"github.com/Oncelane/laneEtcd/src/pkg/laneLog"
)

func TestLock(t *testing.T) {
	wait := sync.WaitGroup{}
	ck.Delete("lock")
	wait.Add(4)
	for i := range 4 {
		go func(routine int) {
			defer wait.Done()
			var (
				id  string
				err error
			)
			for {
				id, err = ck.Lock("lock", 0)
				if err != nil {
					laneLog.Logger.Fatalln(err)
				}
				if id != "" {
					break
				}
				time.Sleep(500 * time.Millisecond)

			}
			laneLog.Logger.Infof("routine[%d] gain lock", routine)
			laneLog.Logger.Infof("routine[%d] do something (sleep 500ms)", routine)
			time.Sleep(time.Millisecond * 500)
			laneLog.Logger.Infof("routine[%d] release the lock", routine)
			ok, err := ck.Unlock("lock", id)
			if err != nil {
				laneLog.Logger.Fatalln(err)
			}
			if ok {
				laneLog.Logger.Infof("routine[%d] success unlock", routine)
			} else {
				laneLog.Logger.Infof("routine[%d] faild  unlock", routine)
			}
		}(i)
	}
	wait.Wait()
}
