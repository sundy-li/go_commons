package mq

import (
	"strings"
	"sync"

	"github.com/Shopify/sarama"
	"github.com/bsm/sarama-cluster"

	"github.com/sundy-li/go_commons/conf"
	"github.com/sundy-li/go_commons/log"
)

type KafkaClusterMQ struct {
	mux           *sync.RWMutex
	addrs         []string
	config        *cluster.Config
	syncProducer  sarama.SyncProducer
	aSyncProducer sarama.AsyncProducer
	producerMsg   chan<- *sarama.ProducerMessage
}

func NewKafkaClusterMQ(cfg *conf.DbConfig) (newKafka *KafkaClusterMQ) {
	addrs := strings.Split(cfg.Host, ",")
	config := cluster.NewConfig()
	config.Consumer.Return.Errors = true
	config.Group.Return.Notifications = true

	return &KafkaClusterMQ{
		addrs:  addrs,
		config: config,
		mux:    new(sync.RWMutex),
	}
}

type ClusterConsumer struct {
	groupId   string
	clustermq *KafkaClusterMQ
	consumer  *cluster.Consumer
	closed    chan struct{}
}

func (k *KafkaClusterMQ) NewClusterConsumer(groupId string, topics []string) (c *ClusterConsumer, err error) {
	consumer, err := cluster.NewConsumer(k.addrs, groupId, topics, k.config)
	if err != nil {
		return nil, err
	}
	c = &ClusterConsumer{
		groupId:   groupId,
		clustermq: k,
		consumer:  consumer,
		closed:    make(chan struct{}),
	}
	return
}

func (c *ClusterConsumer) Receive() chan *sarama.ConsumerMessage {
	buf := make(chan *sarama.ConsumerMessage, 10000)

	go func(buf chan<- *sarama.ConsumerMessage, consumer *cluster.Consumer) {
		for {
			select {
			case msg, more := <-consumer.Messages():
				if !more {
					return
				}

				buf <- msg
				consumer.MarkOffset(msg, "") // mark message as processed
			case _, more := <-consumer.Errors():
				if !more {
					log.Infof("Errors 管道关闭。退出")
					return
				}
			case _, more := <-consumer.Notifications():
				if !more {
					return
				}
			case <-c.closed:
				return
			}
		}
	}(buf, c.consumer)

	return buf
}

func (c *ClusterConsumer) Close() {
	close(c.closed)
}

func (k *KafkaClusterMQ) newSyncProducer() (err error) {
	k.syncProducer, err = sarama.NewSyncProducer(k.addrs, nil)
	return
}

func (k *KafkaClusterMQ) newASyncProducer() (err error) {
	k.aSyncProducer, err = sarama.NewAsyncProducer(k.addrs, nil)
	if err != nil {
		return
	}
	k.producerMsg = k.aSyncProducer.Input()
	return
}

func (k *KafkaClusterMQ) SendString(topic string, msg string, async bool) (err error) {
	if async {
		if k.producerMsg == nil {
			if err = k.newASyncProducer(); err != nil {
				return
			}
		}
	} else {
		if k.syncProducer == nil {
			if err = k.newSyncProducer(); err != nil {
				return
			}
		}
	}

	kafkaMsg := &sarama.ProducerMessage{Topic: topic, Value: sarama.StringEncoder(msg)}
	if async {
		k.producerMsg <- kafkaMsg
	} else {
		_, _, err = k.syncProducer.SendMessage(kafkaMsg)
	}

	return
}
