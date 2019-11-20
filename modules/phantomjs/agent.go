package phantomjs

import (
	"bufio"
	"io"
	"os/exec"
	"strconv"
	"syscall"

	"github.com/sundy-li/go_commons/log"
)

type Monitor interface {
	Register(*Agent)
	Close(*Agent)
}

type Agent struct {
	Id       int
	proxyUrl string
	jsPath   string
	binPath  string
	port     int
	monitor  Monitor
	cmd      *exec.Cmd
	state    int // 1 run 0 down
	stop     chan struct{}
	stoped   chan struct{}
}

func NewAgent(proxyUrl string, binPath string, jsPath string, port int, monitor Monitor) *Agent {
	return &Agent{
		proxyUrl: proxyUrl,
		jsPath:   jsPath,
		binPath:  binPath,
		port:     port,
		monitor:  monitor,
		state:    1,
		stop:     make(chan struct{}),
		stoped:   make(chan struct{}),
	}
}

func (a *Agent) GetPort() int {
	return a.port
}

func (a *Agent) Run() {
	a.monitor.Register(a)
	if err := a.run(); err != nil {
		a.monitor.Close(a)
		return
	}

	go func() {
		defer log.Infof("port %d shutdown", a.port)
		for {
			select {
			case <-a.stoped:
				a.stop <- struct{}{}
				return
			}
		}
	}()

	go a.wait()
}

func (a *Agent) Stop() {
	a.stoped <- struct{}{}
}

func (a *Agent) IsDown() bool {
	return a.state == 0
}

// clean
func (a *Agent) wait() {
	<-a.stop
	if err := a.kill(); err != nil {
		log.Warnf("kill cmd err:%s", err.Error())
	}

	a.state = 0
	a.monitor.Close(a)
	close(a.stop)
	close(a.stoped)
}

func (a *Agent) prettyLog(i io.ReadCloser) error {
	scanner := bufio.NewScanner(i)
	for scanner.Scan() {
		log.Info(scanner.Text())
	}
	return i.Close()
}

func (a *Agent) kill() error {
	if a.cmd.Process != nil {
		// thread unsafe
		return syscall.Kill(a.cmd.Process.Pid, syscall.SIGKILL)
	}

	return nil
}

func (a *Agent) run() error {
	cmd := exec.Command(a.binPath, "--proxy", a.proxyUrl, a.jsPath, strconv.Itoa(a.port))
	a.cmd = cmd

	stdOut, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	// pretty log
	go a.prettyLog(stdOut)

	// cmd run
	go func() {
		err := cmd.Run()
		if err != nil && a.state == 1 {
			// 判断是否为自杀，自杀则不打印该条日志
			log.Warnf("agent port:%d be kill, err:%s", a.port, err.Error())
		}

		if a.state == 1 {
			a.Stop()
		}
	}()
	return nil
}
