package redis

import (
	"sort"
	"testing"
	"time"

	"github.com/wswz/go_commons/libs/assert"
)

func TestUintSet(t *testing.T) {
	set := newUintSet(10)
	assert.False(t, set.contain(1))

	set.insert(1)
	assert.True(t, set.contain(1))

	set.delete(1)
	assert.False(t, set.contain(1))
}

// 测试时需要开启本地的redis-server,并且端口为 6379
func TestMultiMq(t *testing.T) {
	wrongConfig := RedisConfig{
		Host: "127.0.0.1",
		Port: 6666,
	}
	wrongConfigs := make([]RedisConfig, 0, 3)
	wrongConfigs = append(wrongConfigs, wrongConfig, wrongConfig, wrongConfig)
	mq := NewMultiMq(wrongConfigs, time.Second, nil)
	assert.Equals(t, mq.Ping(), uint(0))
	assert.NoNil(t, mq.Push("wrongmq", "val1"))
	_, err := mq.Pop("wrongmq", 0)
	assert.NoNil(t, err)
	assert.Equals(t, err, FAILED_SERVER)
	_, err = mq.Pop("wrongmq", 1)
	assert.StringContains(t, err.Error(), "connection refused")
	assert.NoNil(t, err)
	_, err = mq.Pop("wrongmq", 2)
	assert.NoNil(t, err)
	_, err = mq.Pop("wrongmq", 3)
	assert.NoNil(t, err)
	assert.Equals(t, err, OUT_OF_RANGE)
	err = mq.Push("wrongmq", "val")
	assert.NoNil(t, err)
	assert.Equals(t, err, ALL_FAILED)
	mq.Close()

	rightConfig := RedisConfig{
		Host: "127.0.0.1",
		Port: 6379,
	}
	rightConfigs := make([]RedisConfig, 0, 3)
	rightConfigs = append(rightConfigs, rightConfig, rightConfig, rightConfig)
	mq = NewMultiMq(rightConfigs, time.Second, nil)
	assert.Equals(t, mq.Ping(), uint(3))
	assert.Nil(t, mq.Push("rightmq", "val1"))
	assert.Nil(t, mq.Push("rightmq", "val2"))
	assert.Nil(t, mq.Push("rightmq", "val3"))
	result := make([]string, 0, 3)
	var val string
	for i := 0; i < 3; i++ {
		for err = nil; err == nil; {
			if val, err = mq.Pop("rightmq", uint(i)); err == nil {
				result = append(result, val)
			}
		}
	}
	sort.Strings(result)
	assert.DeepEquals(t, result, []string{"val1", "val2", "val3"})

	_, err = mq.Pop("rightmq", 3)
	assert.NoNil(t, err)
	assert.Equals(t, err, OUT_OF_RANGE)
	mq.Close()
}
