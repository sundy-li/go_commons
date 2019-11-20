package proxy

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/url"
	"strings"
	"sync"

	"github.com/sundy-li/go_commons/log"
)

type Proxy struct {
	once     sync.Once
	inet     []string
	addr     string
	listener net.Listener
}

func NewProxy(inet []string, addr string) *Proxy {
	return &Proxy{
		inet: inet,
		addr: addr,
	}
}

var Nil = errors.New("not found inet")

func (p *Proxy) init() {
	l, err := net.Listen("tcp", p.addr)
	if err != nil {
		log.Error(err)
	}

	p.listener = l
	return
}

func (p *Proxy) GetConn(network, addr string) (net.Conn, error) {
	if len(p.inet) == 0 {
		return nil, Nil
	}

	inet := p.inet[rand.Intn(len(p.inet))]
	return getConnByName(inet, network, addr)
}

func (p *Proxy) Run() error {
	p.once.Do(p.init)

	for {
		client, err := p.listener.Accept()
		if err != nil {
			log.Errorf("listenner err:%s", err.Error())
		}
		go p.handleClientRequest(client)
	}
}

func (p *Proxy) handleClientRequest(conn net.Conn) {
	if conn == nil {
		return
	}
	defer conn.Close()

	var b [1024]byte
	n, err := conn.Read(b[:])
	if err != nil {
		log.Warnf("read conn:%s", err.Error())
		return
	}

	var method, host, address string
	fmt.Sscanf(string(b[:bytes.IndexByte(b[:], '\n')]), "%s%s", &method, &host)
	hostPortURL, err := url.Parse(host)
	if err != nil {
		log.Warnf("parse url:%s err:%s", host, err.Error())
		return
	}

	if hostPortURL.Opaque == "443" { //https访问
		address = hostPortURL.Scheme + ":443"
	} else { //http访问
		if strings.Index(hostPortURL.Host, ":") == -1 { //host不带端口， 默认80
			address = hostPortURL.Host + ":80"
		} else {
			address = hostPortURL.Host
		}
	}

	// 代理拨号
	server, err := p.GetConn("tcp", address)
	if err != nil {
		log.Warnf("get conn address:%s err:%s", address, err.Error())
		return
	}

	if method == "CONNECT" {
		fmt.Fprint(conn, "HTTP/1.1 200 Connection established\r\n\r\n")
	} else {
		server.Write(b[:n])
	}

	//进行转发
	go io.Copy(server, conn)
	io.Copy(conn, server)
}
