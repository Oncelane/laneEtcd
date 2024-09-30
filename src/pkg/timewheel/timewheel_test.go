package timewheel

import (
	"math/rand"
	"testing"
	"time"

	"github.com/Oncelane/laneEtcd/src/pkg/laneLog"
	"github.com/nosixtools/timewheel"
)

//

var mock = make(map[string]string)

func init() {
	laneLog.InitLogger("timeWheel", true, true, false)
}
func TestTimeWheel(t *testing.T) {
	tw := timewheel.New(time.Second, 200)
	tw.Start()
	defer tw.Stop() // Ensure the time wheel stops after the test

	key := "comet0"
	value := "addr"

	// Main task
	watchKey := AddKey(tw, key, value, time.Second*4)

	// mock hearbeat
	MockHearBeat(tw, watchKey, key, value)

	// Assert expected state of mock
	select {}
	laneLog.Logger.Infoln("Finished test for timewheel")
}

func AddKey(tw *timewheel.TimeWheel, key string, value string, timeout time.Duration) (watchdog string) {
	mock[key] = value
	watchdog = "watch:" + key
	tw.AddTask(timeout, -1, key, nil, func(td timewheel.TaskData) {
		tw.RemoveTask(key)
		tw.RemoveTask(watchdog)
		laneLog.Logger.Warnln("end delete key")
	})
	return
}

func MockHearBeat(tw *timewheel.TimeWheel, watchKey string, key string, value string) {
	tw.AddTask(time.Second, -1, watchKey, nil, func(td timewheel.TaskData) {

		// remove delay key
		tw.RemoveTask(key)
		AddKey(tw, key, value, time.Second*4)

		// remove itself, just for safity
		tw.RemoveTask(watchKey)

		// heartbeatnext time
		nextHeartBeat := time.Second * time.Duration(rand.Int()%2+1)
		laneLog.Logger.Infof("下次心跳在 %v", nextHeartBeat)
		if nextHeartBeat > time.Second*4 {
			laneLog.Logger.Infoln("预计超时，触发delete")
		}
		MockHearBeat(tw, watchKey, key, value)
		laneLog.Logger.Infof("keep %s alive", key)
	})
}
