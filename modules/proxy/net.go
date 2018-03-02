package proxy

import (
	"errors"
	"net"
	"strings"
	"time"
)

func getConnByName(name, network, address string) (conn net.Conn, err error) {
	intf, err := net.InterfaceByName(name)
	if err != nil {
		return nil, err
	}

	addrs, err := intf.Addrs()
	if err != nil {
		return nil, err
	}

	if len(addrs) < 1 {
		return nil, errors.New("not addrs")
	}

	var ipv4Addr string
	for _, addr := range addrs {
		addrStr := strings.Split(addr.String(), "/")[0]
		if isIPv4(addrStr) {
			ipv4Addr = addrStr
			break
		}
	}

	ipAddr, err := net.ResolveTCPAddr("tcp", ipv4Addr+":0")
	if err != nil {
		return
	}

	dialer := net.Dialer{
		Timeout:   time.Second * 10,
		LocalAddr: ipAddr,
		KeepAlive: time.Second * 60,
	}

	return dialer.Dial(network, address)
}

func isIPv4(addr string) bool {
	return net.ParseIP(addr).To4() != nil
}
