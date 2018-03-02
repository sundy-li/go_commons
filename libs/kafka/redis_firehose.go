package kafka

import (
	"errors"
	"log"
	"time"

	"github.com/Shopify/sarama"
	"gopkg.in/redis.v5"
)

// 从 redis 的队列获取数据然后放进 Kafka 队列的辅助方法
type TransferConf struct {
	Redis         *redis.Client
	RedisKey      string
	KafkaProducer sarama.AsyncProducer
	KafkaTopic    string
	Size          int64
	IgnoreErr     bool
	IdleSleepSec  int64
	StopChan      chan chan bool
}

const FOREVER = -999
const DEF_SLEEP_SEC = 1

func Transfer(conf *TransferConf) (err error) {
	size := conf.Size
	if size == 0 {
		return errors.New("TransferConf配置中Size不能为0")
	}
	redisConn := conf.Redis
	kafkaConn := conf.KafkaProducer
	runForever := conf.Size == FOREVER
	if conf.StopChan == nil {
		conf.StopChan = make(chan chan bool, 1)
	}
	sleepSec := conf.IdleSleepSec
	if sleepSec == 0 {
		sleepSec = DEF_SLEEP_SEC
	}

	var popRes *redis.StringSliceCmd
	var res []string
	for runForever || size > 0 {
		select {
		case c := <-conf.StopChan:
			log.Printf(conf.KafkaTopic + "收到退出信息,正常退出")
			c <- true
			return nil
		default:
		}
		popRes = redisConn.BRPop(time.Second*time.Duration(sleepSec), conf.RedisKey)
		if err = popRes.Err(); err == nil {
			//插入到 kafka 中
			res, err = popRes.Result()
			if err == nil {
				for i, val := range res {
					if i == 0 {
						continue
					}
					select {
					case e := <-kafkaConn.Errors():
						if e != nil && e.Err != nil {
							err = e.Err
						}
					case kafkaConn.Input() <- &sarama.ProducerMessage{Topic: conf.KafkaTopic, Value: sarama.StringEncoder(val)}:
						log.Printf(conf.KafkaTopic+"成功获取了信息 [%s] 并放到Kafka中", val)
						if !runForever {
							size--
						}
					}
				}
				continue
			}
		}

		if !conf.IgnoreErr {
			return err
		} else if err != nil {
			log.Printf(conf.KafkaTopic+" 错误:%s", err.Error())
		}
	}
	return
}
