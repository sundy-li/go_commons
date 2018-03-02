package proxy

import (
	"fmt"
	"time"

	"go_commons/log"
)

type Account struct {
	Server string
	User   string
	Passwd string
}

type PPTPManager struct {
	accounts []*Account
	prefix   string
	tunnels  []string
	pptp     *PPTP
	flush    *time.Ticker
}

func NewPPTPManager(prefix string, accounts []*Account, pptp *PPTP, flush time.Duration) *PPTPManager {
	return &PPTPManager{
		accounts: accounts,
		prefix:   prefix,
		pptp:     pptp,
		tunnels:  make([]string, 0, len(accounts)),
		flush:    time.NewTicker(flush),
	}
}

func (p *PPTPManager) Run() {
	for i, ac := range p.accounts {
		tunnel := fmt.Sprintf("%s%d", p.prefix, i)
		if err := p.pptp.Create(tunnel, ac.Server, ac.User, ac.Passwd); err != nil {
			log.Warnf("create tunnel:%s account:%+v err:%s", tunnel, ac, err.Error())
			continue
		}

		p.tunnels = append(p.tunnels, tunnel)
	}

	go p.autoRestart()
}

func (p *PPTPManager) autoRestart() {
	for t := range p.flush.C {
		log.Infof("auto restart %s", t.String())
		p.restart()
	}
}

func (p *PPTPManager) restart() {
	for _, tunnel := range p.tunnels {
		if err := p.pptp.ReStart(tunnel); err != nil {
			log.Warnf("restart tunnel:%s err:%s", tunnel, err.Error())
		}
	}
}

func (p *PPTPManager) GetInet() []string {
	return p.tunnels
}

// 手动销毁
func (p *PPTPManager) Stop() {
	p.destroy()
}

// 销毁tunnels
func (p *PPTPManager) destroy() {
	p.flush.Stop()
	for _, tunnel := range p.tunnels {
		p.pptp.POff(tunnel)
		if err := p.pptp.Delete(tunnel); err != nil {
			log.Warnf("delete tunnel:%s err:%s", tunnel, err.Error())
		}
	}
}
