package redis

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"gopkg.in/redis.v5"
)

var Nil = redis.Nil

type MultiMq struct {
	redises        []*redis.Client
	failedClients  *uintSet
	checkTicker    *time.Ticker
	idxPartitioner Partition
}

type RedisConfig struct {
	Host string
	Port int
}

func NewMultiMq(configs []RedisConfig, checkInterval time.Duration, partitioner Partition) *MultiMq {
	var mq MultiMq
	mq.redises = make([]*redis.Client, 0, len(configs))
	mq.failedClients = newUintSet(len(configs))
	mq.idxPartitioner = partitioner
	if mq.idxPartitioner == nil {
		mq.idxPartitioner = NewStepPartitioner(uint(len(configs)))
	}
	for _, conf := range configs {
		client := redis.NewClient(&redis.Options{
			Addr:    fmt.Sprintf("%s:%d", conf.Host, conf.Port),
			Network: "tcp",
		})
		mq.redises = append(mq.redises, client)
	}
	mq.checkTicker = time.NewTicker(checkInterval)
	go mq.checkStat()
	return &mq
}

// 返回成功ping通的redis的数目
func (mq *MultiMq) Ping() uint {
	var count uint = 0
	for _, cli := range mq.redises {
		if cli.Ping().Val() == "PONG" {
			count++
		}
	}
	return count
}

func (mq *MultiMq) Push(mqName string, val string) error {
	idx, err := mq.idxPartitioner.GetIndex(mq.failedClients)
	if err != nil {
		return err
	}
	cmd := mq.redises[idx].RPush(mqName, val)
	if cmd.Err() != nil {
		mq.failedClients.insert(idx)
		return cmd.Err()
	} else {
		return nil
	}
}

func (mq *MultiMq) Pop(mqName string, cliId uint) (string, error) {
	if cliId >= uint(len(mq.redises)) {
		return "", OUT_OF_RANGE
	}
	isFailed := mq.failedClients.contain(cliId)
	if isFailed {
		return "", FAILED_SERVER
	}
	cmd := mq.redises[cliId].LPop(mqName)
	if cmd.Err() != nil && cmd.Err() != redis.Nil {
		mq.failedClients.insert(cliId)
		return "", cmd.Err()
	} else {
		return cmd.Val(), cmd.Err()
	}
}

func (mq *MultiMq) checkStat() {
	for _ = range mq.checkTicker.C {
		failedClientsCopied := mq.failedClients.copyToSlice()
		for _, offlineCli := range failedClientsCopied {
			stat := mq.redises[offlineCli].Ping()
			if stat.Val() == "PONG" { //离线的redis服务器已恢复
				mq.failedClients.delete(offlineCli)
			}
		}
	}
}

// 关闭 checkTicker，否则 go checkStat() goroutine 会一直运行
func (mq *MultiMq) Close() {
	mq.checkTicker.Stop()
}

type uintSet struct {
	inner map[uint]struct{}
	lock  sync.RWMutex
}

func newUintSet(capacity int) *uintSet {
	set := make(map[uint]struct{}, capacity)
	return &uintSet{inner: set}
}

func (set *uintSet) contain(key uint) bool {
	set.lock.RLock()
	_, ok := set.inner[key]
	set.lock.RUnlock()
	return ok
}

func (set *uintSet) insert(key uint) {
	set.lock.Lock()
	set.inner[key] = struct{}{}
	set.lock.Unlock()
}

func (set *uintSet) delete(key uint) {
	set.lock.Lock()
	delete(set.inner, key)
	set.lock.Unlock()
}

func (set *uintSet) copyToSlice() []uint {
	set.lock.RLock()
	s := make([]uint, 0, len(set.inner))
	for k, _ := range set.inner {
		s = append(s, k)
	}
	set.lock.RUnlock()
	return s
}

type Partition interface {
	GetIndex(*uintSet) (uint, error)
}

type StepPartitioner struct {
	step       uint64
	upperBound uint
}

func NewStepPartitioner(upperBound uint) *StepPartitioner {
	return &StepPartitioner{
		step:       0,
		upperBound: upperBound,
	}
}

func (p *StepPartitioner) GetIndex(failedClients *uintSet) (uint, error) {
	for i := uint(0); i < p.upperBound; i++ {
		idx := uint(p.step) % p.upperBound
		p.step++
		if !failedClients.contain(idx) {
			return idx, nil
		}
	}
	return 0, ALL_FAILED
}

type PriorityPartitioner struct {
	upperBound uint
}

func NewPriorityPartitioner(upperBound uint) *PriorityPartitioner {
	return &PriorityPartitioner{
		upperBound: upperBound,
	}
}

func (p *PriorityPartitioner) GetIndex(failedClients *uintSet) (uint, error) {
	for i := uint(0); i < p.upperBound; i++ {
		if !failedClients.contain(i) {
			return i, nil
		}
	}
	return 0, ALL_FAILED
}

// 错误定义
var ALL_FAILED = errors.New("all the redis servers failed.")
var FAILED_SERVER = errors.New("failed server")
var OUT_OF_RANGE = errors.New("out of range")
