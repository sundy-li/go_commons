package proxy

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"testing"
)

// 测试出口网卡是否为选定的网卡外网IP
func TestGetConnByName(t *testing.T) {
	client := new(http.Client)
	client.Transport = &http.Transport{
		Dial: func(network, addr string) (net.Conn, error) {
			return getConnByName("ppp0", network, addr)
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
