package client_test

import (
	"sync"
	"testing"
	"time"

	"github.com/Oncelane/laneEtcd/src/pkg/laneLog"
)

func TestClient(t *testing.T) {
	ch := make(chan int)
	targetIndex := 10
	var wait sync.WaitGroup
	wait.Add(11)
	for id := range 10 {
		go func(id int) {
			lastIndex := 0
			defer wait.Done()
			for {
				if lastIndex == targetIndex {
					laneLog.Logger.Infof("id:%d success", id)
					return
				}
				if lastIndex > targetIndex {
					laneLog.Logger.Infof("id:%d faild?", id)
				}
				select {
				case index := <-ch:
					lastIndex = index
					laneLog.Logger.Infof("id:%d trigger by index:%d", id, index)
					continue
				case <-time.After(time.Millisecond * 500):
					laneLog.Logger.Infof("id:%d out of time", id)
					return
				}
			}
		}(id)
	}
	go func() {
		defer wait.Done()
		lastIndex := 1
		for range 15 {
			time.Sleep(time.Millisecond * 40)
			for {
				select {
				case ch <- lastIndex:
					laneLog.Logger.Infoln("producer gen", lastIndex)
					lastIndex++
				default:
					laneLog.Logger.Infoln("producer think await all routine", lastIndex)
					goto BREAK
				}
			}
		BREAK:
			laneLog.Logger.Infoln("producer quit")
		}
	}()
	wait.Wait()
}
