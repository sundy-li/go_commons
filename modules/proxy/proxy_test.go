package proxy

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"
)

func TestProxy(t *testing.T) {
	wg := new(sync.WaitGroup)
	proxy := NewProxy([]string{"en0", "ppp0"}, ":6008")
	wg.Add(1)
	go func() {
		wg.Done()
		if err := proxy.Run(); err != nil {
			t.Fatal(err)
		}
	}()

	// wait for server
	wg.Wait()
	time.Sleep(time.Microsecond * 50)
	for i := 0; i < 10; i++ {
		client := &http.Client{}
		client.Transport = &http.Transport{
			Proxy: func(r *http.Request) (*url.URL, error) {
				return url.Parse("http://127.0.0.1:6008")
			},
		}

		resp, err := client.Get("http://ip.chinaz.com/getip.aspx")
		if err != nil {
			t.Fatal(err)
		}

		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		fmt.Println(string(data))
	}
}
