package laneConfig

import (
	"log"
	"os"

	"github.com/Oncelane/laneEtcd/src/pkg/laneLog"

	yaml "gopkg.in/yaml.v3"
)

type LaneConfig interface {
	Default()
}

// TODO etcd dynamic config

func WriteRemote(conf LaneConfig) error {

	return nil
}

// TODO etcd dynamic config
func ReadRemote(conf LaneConfig) {

}

func Init(Path string, conf LaneConfig) {
	_, err := os.Stat(Path)
	if err != nil {
		if os.IsNotExist(err) {
			conf.Default()
			WriteLocal(Path, conf)
			laneLog.Logger.Warnf("please check for the %s if needed to be modified, then run again\n", Path)
		} else {
			laneLog.Logger.Fatalln("config wrong err:", err)
		}
	}
	ReadLocal(Path, conf)
}

func WriteLocal(Path string, conf LaneConfig) error {
	out, err := yaml.Marshal(conf)
	if err != nil {
		laneLog.Logger.Fatalln("failed to marshal config", Path, ":", err)
		return err
	}

	err = os.WriteFile(Path, out, 0644)
	if err != nil {
		laneLog.Logger.Fatalln("failed to write ", Path, err)
		return err
	}
	return nil
}

func ReadLocal(Path string, conf LaneConfig) error {
	log.Println("read from ", Path)
	data, err := os.ReadFile(Path)
	if err != nil {
		laneLog.Logger.Fatalln("config.yaml does not exist")
	}

	err = yaml.Unmarshal(data, conf)
	if err != nil {
		laneLog.Logger.Fatalln("can't not read config.yml")
	}
	return err
}
