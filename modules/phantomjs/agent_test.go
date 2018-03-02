package phantomjs

import (
	"testing"
	"time"
)

func TestAgent(t *testing.T) {
	proxy := "http://127.0.0.1:8080"
	proxy = ""
	ports := []int{8990, 8991, 9999, 9989}
	monitor := NewPhantomJs(
		proxy,
		"/usr/local/bin/phantomjs",
		"./web_brower_service.js",
		ports,
	)
	monitor.Run(3)
	time.Sleep(time.Second * 10)
	monitor.Shutdown()
}
