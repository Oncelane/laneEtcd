package main

import (
	"net/http"

	"github.com/Oncelane/laneEtcd/src/client"
	"github.com/Oncelane/laneEtcd/src/pkg/laneConfig"
	"github.com/gin-gonic/gin"
)

var ck *client.Clerk

func init() {
	conf := laneConfig.Clerk{}
	laneConfig.Init("config.yml", &conf)
	// laneLog.Logger.Debugln("check conf", conf)
	ck = client.MakeClerk(conf)
}

func main() {
	r := gin.Default()
	r.GET("/keys", getKeys)
	r.Run(":9292")
}

func getKeys(c *gin.Context) {
	switch c.Query("type") {
	case "all":
	case "page":

	default:
		c.JSON(http.StatusOK, gin.H{"msg": "wrong type"})
	}
}
