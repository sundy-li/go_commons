package redis

import (
	"fmt"
	"strconv"

	"go_commons/conf"

	"gopkg.in/redis.v5"
)

func ConnectRedis(host string, port int, db int) (client *redis.Client, err error) {
	p := strconv.Itoa(port)
	return NewClient(host, p, db)
}

func NewClient(host, port string, db int) (client *redis.Client, err error) {
	client = redis.NewClient(&redis.Options{
		Addr: host + ":" + port,
		DB:   db, // use default DB
	})

	_, err = client.Ping().Result()
	return
}

func NewClientFromConfig(config *conf.DbConfig) (client *redis.Client, err error) {
	db, err := strconv.Atoi(config.Db)
	client = redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%d", config.Host, config.Port),
		DB:   db, // use default DB
	})
	_, err = client.Ping().Result()
	return
}
