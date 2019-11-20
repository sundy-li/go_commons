package mq

import (
	"sync"

	"github.com/sundy-li/go_commons/conf"
	"github.com/sundy-li/go_commons/libs/mq"
)

var (
	redisMutex sync.Mutex
	redises    = make(map[string]*mq.RedisMQ)
)

func GetRedisCluster(name string) (producer *mq.RedisMQ) {
	if _, ok := redises[name]; !ok {
		redisMutex.Lock()
		defer redisMutex.Unlock()
		cfg := conf.GetResConfig().Redis[name]
		redises[name] = mq.NewRedisMQ(cfg)
	}
	return redises[name]
}
