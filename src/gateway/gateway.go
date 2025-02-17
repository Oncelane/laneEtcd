package gateway

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Oncelane/laneEtcd/src/client"
	"github.com/Oncelane/laneEtcd/src/common"
	"github.com/Oncelane/laneEtcd/src/pkg/laneConfig"
	"github.com/gin-gonic/gin"
)

// func init() {
// 	conf := laneConfig.Gateway{}
// 	laneConfig.Init("config.yml", &conf)
// 	// laneLog.Logger.Debugln("check conf", conf)
// 	ck = client.MakeClerk(conf.Clerk)
// }

// func main() {
// 	r := gin.Default()
// 	r.GET("/keys", getKeys)
// 	r.Run(":9292")
// }

type Gateway struct {
	ck   *client.Clerk
	conf laneConfig.Gateway
}

func NewGateway(conf laneConfig.Gateway) *Gateway {
	return &Gateway{
		ck:   client.MakeClerk(conf.Clerk),
		conf: conf,
	}
}

// realurl = baseUrl + '/keys' | '/put' act...
// default baseUrl = '/laneEtcd'
func (g *Gateway) Run() error {
	r := gin.Default()
	r.GET(g.conf.BaseUrl+"/keys", g.keys)
	r.GET(g.conf.BaseUrl+"/key", g.get)
	r.GET(g.conf.BaseUrl+"/keysWithPrefix", g.getWithPrefix)
	r.GET(g.conf.BaseUrl+"/kvs", g.kvs)
	r.POST(g.conf.BaseUrl+"/put", g.put)
	r.POST(g.conf.BaseUrl+"/putCAS", g.putCAS)
	r.DELETE(g.conf.BaseUrl+"/key", g.del)
	r.DELETE(g.conf.BaseUrl+"/keysWithPrefix", g.delWithPrefix)
	return r.Run(g.conf.Port)
}

func (g *Gateway) LoadRouter(r *gin.Engine) {
	r.GET(g.conf.BaseUrl+"/keys", g.keys)
	r.GET(g.conf.BaseUrl+"/key", g.get)
	r.GET(g.conf.BaseUrl+"/keysWithPrefix", g.getWithPrefix)
	r.GET(g.conf.BaseUrl+"/kvs", g.kvs)
	r.POST(g.conf.BaseUrl+"/put", g.put)
	r.POST(g.conf.BaseUrl+"/putCAS", g.putCAS)
	r.DELETE(g.conf.BaseUrl+"/key", g.del)
	r.DELETE(g.conf.BaseUrl+"/keysWithPrefix", g.delWithPrefix)
}

func (g *Gateway) keys(c *gin.Context) {
	var pairs []Pair
	switch c.Query("type") {
	case "", "all":
		cpairs, err := g.ck.Keys()
		if err != nil {
			c.JSON(http.StatusOK, common.Respond{Code: http.StatusOK, Data: gin.H{"msg": err.Error()}})
			return
		}
		pairs = make([]Pair, len(cpairs))
		for i := range cpairs {
			pairs[i].Key = cpairs[i].Key
			pairs[i].DeadTime = cpairs[i].Entry.DeadTime
		}
		c.JSON(http.StatusOK, common.Respond{Code: http.StatusOK, Data: pairs})

	case "page":

	default:
		c.JSON(http.StatusOK, common.Respond{Code: http.StatusOK, Data: gin.H{"msg": "wrong type"}})
	}
}

func (g *Gateway) kvs(c *gin.Context) {
	var pairs []Pair
	switch c.Query("type") {
	case "", "all":
		cpairs, err := g.ck.KVs()
		if err != nil {
			c.JSON(http.StatusOK, common.Respond{Code: http.StatusOK, Data: gin.H{"msg": err.Error()}})
			return
		}
		pairs = make([]Pair, len(cpairs))
		for i := range cpairs {
			pairs[i].Key = cpairs[i].Key
			pairs[i].Value = common.BytesToString(cpairs[i].Entry.Value)
			pairs[i].DeadTime = cpairs[i].Entry.DeadTime
		}
		c.JSON(http.StatusOK, common.Respond{Code: http.StatusOK, Data: pairs})

	case "page":
		c.JSON(http.StatusOK, common.Respond{Code: http.StatusOK, Data: gin.H{"msg": "wrong type"}})

	default:
	}
}

type Pair struct {
	Key      string
	Value    string
	DeadTime int64
}

func (g *Gateway) put(c *gin.Context) {
	var pairs []Pair
	err := c.ShouldBind(&pairs)
	if err != nil {
		c.JSON(http.StatusOK, common.Respond{Code: http.StatusOK, Data: gin.H{"msg": err.Error()}})
		return
	}
	var ttl time.Duration
	pipe := g.ck.Pipeline()

	for i := range pairs {
		if pairs[i].DeadTime == 0 {
			ttl = 0
		} else {
			deadTimestamp := time.UnixMilli(pairs[i].DeadTime)
			if time.Now().After(deadTimestamp) {
				c.JSON(http.StatusOK, common.Respond{Code: http.StatusOK, Data: gin.H{"msg": "ok(ignore)"}})
				continue
			}
			ttl = time.Until(deadTimestamp)
		}
		pipe.Put(pairs[i].Key, common.StringToBytes(pairs[i].Value), ttl)
	}
	err = pipe.Exec()
	if err != nil {
		c.JSON(http.StatusOK, common.Respond{Code: http.StatusOK, Data: gin.H{"msg": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, common.Respond{Code: http.StatusOK, Data: gin.H{"msg": "ok"}})

}

type PairCAS struct {
	Key      string
	Value    string
	OriValue string
	DeadTime int64
}

func (g *Gateway) putCAS(c *gin.Context) {
	var pair PairCAS
	err := c.ShouldBind(&pair)
	if err != nil {
		c.JSON(http.StatusOK, common.Respond{Code: http.StatusOK, Data: gin.H{"msg": err.Error()}})
		return
	}
	var ttl time.Duration
	if pair.DeadTime == 0 {
		ttl = 0
	} else {
		deadTimestamp := time.UnixMilli(pair.DeadTime)
		if time.Now().After(deadTimestamp) {
			c.JSON(http.StatusOK, common.Respond{Code: http.StatusOK, Data: gin.H{"msg": "ok(ignore)"}})
			return
		}
		ttl = time.Until(deadTimestamp)
	}
	ok, err := g.ck.CAS(pair.Key, common.StringToBytes(pair.OriValue), common.StringToBytes(pair.Value), ttl)
	if err != nil {
		c.JSON(http.StatusOK, common.Respond{Code: http.StatusOK, Data: gin.H{"msg": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, common.Respond{Code: http.StatusOK, Data: gin.H{"msg": fmt.Sprintf("%t", ok)}})
}

func (g *Gateway) del(c *gin.Context) {
	key := c.Query("key")
	err := g.ck.Delete(key)
	if err != nil {
		c.JSON(http.StatusOK, common.Respond{Code: http.StatusOK, Data: gin.H{"msg": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, common.Respond{Code: http.StatusOK, Data: gin.H{"msg": "ok"}})
}

func (g *Gateway) delWithPrefix(c *gin.Context) {
	prefix := c.Query("prefix")
	err := g.ck.DeleteWithPrefix(prefix)
	if err != nil {
		c.JSON(http.StatusOK, common.Respond{Code: http.StatusOK, Data: gin.H{"msg": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, common.Respond{Code: http.StatusOK, Data: gin.H{"msg": "ok"}})
}

func (g *Gateway) get(c *gin.Context) {
	key := c.Query("key")
	value, err := g.ck.Get(key)
	if err != nil {
		c.JSON(http.StatusOK, common.Respond{Code: http.StatusOK, Data: gin.H{"msg": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, common.Respond{Code: http.StatusOK, Data: gin.H{"value": Pair{
		Value: common.BytesToString(value),
	}, "msg": "ok"}})
}

func (g *Gateway) getWithPrefix(c *gin.Context) {
	prefix := c.Query("prefix")
	value, err := g.ck.GetWithPrefix(prefix)
	if err != nil {
		c.JSON(http.StatusOK, common.Respond{Code: http.StatusOK, Data: gin.H{"msg": err.Error()}})
		return
	}
	pairs := make([]Pair, len(value))
	for i := range value {
		pairs[i].Value = common.BytesToString(value[i])
	}
	c.JSON(http.StatusOK, common.Respond{Code: http.StatusOK, Data: gin.H{"value": pairs, "msg": "ok"}})
}
