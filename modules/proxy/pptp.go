package proxy

import (
	"os/exec"
	"time"

	"github.com/sundy-li/go_commons/log"
)

// pptp util

// please use root account
type PPTP struct {
	pptpsetup string
	poff      string
	pon       string
}

func NewPPTP(pptpsetup string, poff string, pon string) *PPTP {
	return &PPTP{
		pptpsetup: pptpsetup,
		poff:      poff,
		pon:       pon,
	}
}

func (p *PPTP) Delete(tunnel string) error {
	cmd := exec.Command(p.pptpsetup, "--delete", tunnel)
	log.Infof("%s --delete %s", p.pptpsetup, tunnel)
	return cmd.Run()
}

func (p *PPTP) Create(tunnel, server, user, passwd string) error {
	cmd := exec.Command(p.pptpsetup, "--create", tunnel,
		"--server", server, "--username", user,
		"--password", passwd, "--encrypt", "--start")

	log.Infof("%s --create %s --server %s --username %s --password %s --encrypt --start",
		p.pptpsetup, tunnel, server, user, passwd)
	return cmd.Run()
}

func (p *PPTP) ReStart(tunnel string) error {
	p.POff(tunnel)
	time.Sleep(time.Second * 3)
	return p.POn(tunnel)
}

func (p *PPTP) POn(tunnel string) error {
	cmd := exec.Command(p.pon, tunnel)
	log.Infof("%s %s", p.pon, tunnel)
	return cmd.Run()
}

func (p *PPTP) POff(tunnel string) error {
	cmd := exec.Command(p.poff, tunnel)
	log.Infof("%s %s", p.poff, tunnel)
	return cmd.Run()
}
