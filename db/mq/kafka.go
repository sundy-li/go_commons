package mq

import (
	"sync"

	"github.com/wswz/go_commons/conf"
	"github.com/wswz/go_commons/libs/mq"
)

var (
	kafkaMutex sync.Mutex
	kafkas     = make(map[string]*mq.KafkaClusterMQ)
)

func GetKafkaCluster(name string) (producer *mq.KafkaClusterMQ) {
	if _, ok := kafkas[name]; !ok {
		kafkaMutex.Lock()
		defer kafkaMutex.Unlock()
		cfg := conf.GetResConfig().Kafka[name]
		kafkas[name] = mq.NewKafkaClusterMQ(cfg)
	}
	return kafkas[name]
}
