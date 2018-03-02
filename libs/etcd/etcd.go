package etcd

import (
	"context"
	"strings"
	"time"

	"github.com/coreos/etcd/clientv3"

	"github.com/wswz/go_commons/conf"
)

type EtcdDb struct {
	client *clientv3.Client
}

func NewEtcd(cfg *conf.DbConfig) (*EtcdDb, error) {
	addrs := strings.Split(cfg.Host, ",")
	config := clientv3.Config{
		Endpoints: addrs,
	}
	et := new(EtcdDb)
	client, err := clientv3.New(config)
	if err != nil {
		return nil, err
	}
	et.client = client
	return et, nil
}

func (et *EtcdDb) PutTTL(key string, value string, ttl time.Duration) error {
	g, err := et.client.Grant(context.TODO(), int64(ttl/time.Second))
	if err != nil {
		return err
	}
	_, err = et.client.Put(context.TODO(), key, value, clientv3.WithLease(g.ID))
	if err != nil {
		return err
	}
	return nil
}

func (et *EtcdDb) Put(key string, value string) error {
	_, err := et.client.Put(context.TODO(), key, value)
	if err != nil {
		return err
	}
	return nil
}

func (et *EtcdDb) Get(key string) (string, error) {
	resp, err := et.client.Get(context.TODO(), key)
	if err != nil {
		return "", err
	}
	if len(resp.Kvs) == 0 {
		return "", nil
	}
	val := resp.Kvs[0].Value
	return string(val), nil

}

func (et *EtcdDb) WatchPrefix(prefix string) clientv3.WatchChan {
	ch := et.client.Watch(context.TODO(), prefix, clientv3.WithPrefix())
	return ch
}

func (et *EtcdDb) GetPrefix(key string) (map[string]string, error) {
	resp, err := et.client.Get(context.TODO(), key, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	mp := make(map[string]string)
	for _,kv := range resp.Kvs{
		mp[string(kv.Key)] = string(kv.Value)
	}
	return mp, nil

}