package conf

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/wswz/go_commons/log"
	"github.com/wswz/go_commons/utils"

	"github.com/coreos/etcd/clientv3"
)

const (
	ConfxPrefix = "/confx/"
)

type Watcher func(conf string)

type ServerResp struct {
	Error  int    `json:"error"`
	Result string `json:"result"`
}

type ConfigCenter interface {
	Conf() (conf string, err error)
	Watch(watcher Watcher)
}

type EtcdConfigCenter struct {
	addr   string
	proj   string
	etcds  string
	client *clientv3.Client
}

func NewConfigCenter(addr, proj string) ConfigCenter {
	center := &EtcdConfigCenter{
		addr: addr,
		proj: proj,
	}
	if err := center.init(); err != nil {
		panic(err)
	}

	return center
}

func (e *EtcdConfigCenter) init() error {
	err := e.getEtcds()
	if err != nil {
		return err
	}
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   strings.Split(e.etcds, ","),
		DialTimeout: 5 * time.Second,
	})
	e.client = cli
	return err
}

func (e *EtcdConfigCenter) getEtcds() error {
	req, _ := http.NewRequest("GET", e.addr, nil)
	_, data, err := utils.RequestData(req)
	if err != nil {
		return err
	}
	resp := ServerResp{}
	err = json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	e.etcds = resp.Result
	return nil
}

func (e *EtcdConfigCenter) key() string {
	return ConfxPrefix + e.proj
}

func (e *EtcdConfigCenter) Conf() (conf string, err error) {
	ret, err := e.client.Get(context.Background(), e.key(), clientv3.WithLimit(1))
	if err != nil {
		return
	}
	if ret.Count < 1 {
		err = errors.New(fmt.Sprintf("proj: %s do not configured", e.proj))
	} else {
		conf = string(ret.Kvs[0].Value)
	}
	return
}

func (e *EtcdConfigCenter) Watch(watcher Watcher) {
	go func() {
		ch := e.client.Watch(context.Background(), e.key())
		for {
			select {
			case resp := <-ch:
				if resp.Err() != nil {
					log.Warn(resp.Err().Error())
				} else if resp.Canceled {
					goto END
				} else {
					for _, event := range resp.Events {
						watcher(string(event.Kv.Value))
					}
				}
			}
		}
	END:
	}()
}
