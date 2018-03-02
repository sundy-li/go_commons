package etcd

import (
	"testing"
	"go_commons/conf"
	"fmt"
	"time"
	"sync"
)

var et *EtcdDb

func init()  {
	et,_ = NewEtcd(&conf.DbConfig{
		Host:"s3:4010s3:4011,s3:4012",
	})
}

func TestEtcdDb_Put(t *testing.T) {
	et.Put("test","aaa")
	fmt.Println(et.Get("test"))
}

func TestEtcdDb_WatchPrefix(t *testing.T) {
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		for c := range et.WatchPrefix("aaa") {
			fmt.Println(c.Events[0].Kv.Key)
		}
		wg.Done()
	}()
	et.PutTTL("aaa","ddd",time.Second*5)
	et.PutTTL("aaag","ddd",time.Second*5)
	wg.Wait()
}

func TestEtcdDb_PutTTL(t *testing.T) {
	et.PutTTL("test","aaa",time.Second*2)
	time.Sleep(time.Second*3)
	fmt.Println(et.Get("test"))
}