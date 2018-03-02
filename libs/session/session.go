package session

import (
	"log"
	"net/http"
)

type Session struct {
	data      map[string]interface{}
	sessionId string
	provider  Provider

	ExpireSeconds int
}

func NewSession(sid string, mp map[string]interface{}, p Provider) *Session {
	return &Session{
		sessionId: sid,
		data:      mp,
		provider:  p,
	}
}

func (this *Session) Get(key string) interface{} {
	return this.data[key]
}

func (this *Session) Set(key string, value interface{}) {
	this.data[key] = value
}

func (this *Session) Destory() {
	this.data = nil
	this.provider.Delete(this.sessionId)
}

func (this *Session) Release(w http.ResponseWriter) {
	err := this.provider.Release(this, w)
	if err != nil {
		log.Fatalf("release session %s error %s", this.sessionId, err.Error())
	}
}
