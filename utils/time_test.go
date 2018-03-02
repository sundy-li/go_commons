package utils

import (
	"fmt"
	"testing"
	"time"
)

func TestNow(t *testing.T) {
	for i := 1; i <= 10; i++ {
		fmt.Println(now.Unix())
		time.Sleep(time.Second)
	}
}
