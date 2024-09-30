package timewheel_test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/Oncelane/laneEtcd/src/pkg/laneLog"
	"github.com/rfyiamcool/go-timewheel"
)

var mock = make(map[string]string)
var mockstop = false

func TestTimewheel(t *testing.T) {
	tw, _ := timewheel.NewTimeWheel(time.Millisecond, 100)
	tw.Start()
	laneLog.Logger.Infoln("start counting")
	key := "comet"
	value := "addr"
	MockHeartBeat(tw, key, value, time.Millisecond*500)
}
func AddKey(tw *timewheel.TimeWheel, key string, value string, timeout time.Duration) *timewheel.Task {
	mock[key] = value
	task := tw.Add(timeout, func() {
		delete(mock, key)
		laneLog.Logger.Warnln("delete key", key)
		mockstop = true
	})
	return task
}

func MockHeartBeat(tw *timewheel.TimeWheel, key, value string, timeout time.Duration) {
	task := AddKey(tw, key, value, time.Millisecond*500)
	for {
		// mock的部分
		mockTImeout := time.Millisecond * time.Duration((rand.Int()%600 + 1))
		laneLog.Logger.Infoln("next heartbeat in", mockTImeout, "ms")
		time.Sleep(mockTImeout)

		// function
		if mockstop {
			return
		}
		laneLog.Logger.Infoln("heartbeat")
		err := tw.Remove(task)
		if err != nil {
			laneLog.Logger.Infoln(err)
		}

		task = AddKey(tw, key, value, timeout)
	}
}
