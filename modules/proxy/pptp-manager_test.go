package proxy

import (
	"testing"
	"time"
)

func TestPPTPManager(t *testing.T) {
	accounts := []*Account{
		&Account{
			User:   "007",
			Passwd: "007",
			Server: "111.73.46.171",
		},
	}

	pptp := NewPPTP("/usr/sbin/pptpsetup", "/usr/bin/poff", "/usr/bin/pon")
	pptpManager := NewPPTPManager("ppp", accounts, pptp, time.Second*5)
	pptpManager.Run()
	time.Sleep(time.Second * 10)
	pptpManager.Stop()
}
