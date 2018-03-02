package phantomjs

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"go_commons/log"
)

type PhantomJS struct {
	proxy           string
	binPath         string
	jsPath          string
	ports           []int
	stop            bool
	mu              *sync.RWMutex
	currentPortPool map[int]*Agent
}

func NewPhantomJs(proxy, binPath, jsPath string, ports []int) *PhantomJS {
	if len(ports) < 1 {
		log.Warn("ports size is zero,default to 8890")
		ports = []int{8890}
	}
	return &PhantomJS{
		proxy:           proxy,
		binPath:         binPath,
		jsPath:          jsPath,
		ports:           ports,
		mu:              new(sync.RWMutex),
		currentPortPool: make(map[int]*Agent),
	}
}

func (p *PhantomJS) Run(agentNum int) {
	for i := 0; i < agentNum; i++ {
		agent, err := p.NewAgent()
		if err != nil {
			log.Errorf("error in start agent %s", err.Error())
			continue
		}
		agent.Run()
	}
}

func (p *PhantomJS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.Write([]byte("just support GET method"))
		return
	}

	value := r.URL.Query()

	port := p.randomServ()
	if port == 0 {
		w.Write([]byte("not available server"))
		return
	}

	url := "http://127.0.0.1:" + strconv.Itoa(port) + "?" + value.Encode()
	resp, err := http.Get(url)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	defer resp.Body.Close()
	_, _ = io.Copy(w, resp.Body)
	return
}

func (p *PhantomJS) randomServ() (port int) {
	var agent *Agent
	p.mu.RLock()
	defer p.mu.RUnlock()
	for port, agent = range p.currentPortPool {
		if !agent.IsDown() {
			return
		}
	}
	return
}

func (p *PhantomJS) NewAgent() (agent *Agent, err error) {
	if p.stop {
		err = errors.New("the pool shutdown !!!")
		return
	}

	for _, port := range p.ports {
		_, ok := p.currentPortPool[port]
		if ok {
			continue
		}

		agent = NewAgent(p.proxy, p.binPath, p.jsPath, port, p)
		return
	}

	err = errors.New("not any more port can use")
	return
}

func (p *PhantomJS) Register(agent *Agent) {
	p.mu.Lock()
	p.currentPortPool[agent.GetPort()] = agent
	p.mu.Unlock()
}

func (p *PhantomJS) Close(agent *Agent) {
	p.mu.Lock()
	if _, ok := p.currentPortPool[agent.GetPort()]; ok {
		delete(p.currentPortPool, agent.GetPort())
	}
	p.mu.Unlock()
	return
}

func (p *PhantomJS) Count() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.currentPortPool)
}

func (p *PhantomJS) Shutdown() {
	p.stop = true
	p.mu.Lock()
	for port, agent := range p.currentPortPool {
		log.Info(port, "will be kill")
		agent.Stop()
	}
	p.mu.Unlock()

	for {
		if p.Count() == 0 {
			break
		}
		time.Sleep(time.Second * 1)
	}
	log.Infof("shutdown and now pool length:%d", p.Count())
}
