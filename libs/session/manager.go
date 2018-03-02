package session

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"time"
)

const (
	ProviderRedisType  = "redis"
	MemoryProviderType = "memory"
)

//session manager
type Manager struct {
	Provider Provider
	Config   *Config
}

type Config struct {
	Provider string

	GcLife        int
	MaxCookieLife int
	CookieName    string
}

type Provider interface {
	Init(sid string) *Session
	Gc()
	Read(sid string) (*Session, error)
	Release(session *Session, w http.ResponseWriter) error
	Delete(sid string)
}

func (m *Manager) Start(w http.ResponseWriter, r *http.Request) (session *Session) {
	//判断是否有sessionId
	c, err := r.Cookie(m.Config.CookieName)
	if c == nil || err != nil {
		//set sessionId
		sessionId := GensessionId(r)
		m.SetCookie(sessionId, w, r)
		session = m.GenSession(sessionId)
	} else {
		sessionId, _ := url.QueryUnescape(c.Value)
		session, err = m.Provider.Read(sessionId)
		if err != nil || session == nil {
			session = m.GenSession(sessionId)
		}
	}
	return
}

func (m *Manager) GenSession(sessionId string) (session *Session) {
	session = m.Provider.Init(sessionId)
	session.ExpireSeconds = m.Config.MaxCookieLife
	return
}

func (m *Manager) SetCookie(sessionId string, w http.ResponseWriter, r *http.Request) {
	cookie := &http.Cookie{
		Name:     m.Config.CookieName,
		Value:    sessionId,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		MaxAge:   m.Config.MaxCookieLife,
	}
	if cookie.MaxAge > 0 {
		cookie.Expires = time.Now().Add(time.Duration(cookie.MaxAge) * time.Second)
	} else {
		cookie.Expires = time.Now().Add(180 * 365 * 24 * time.Hour)
	}
	http.SetCookie(w, cookie)
	r.AddCookie(cookie)
}

func (m *Manager) Gc() {
	m.Provider.Gc()
	time.AfterFunc(time.Duration(m.Config.GcLife)*time.Second, func() {
		m.Gc()
	})
}

func GensessionId(r *http.Request) string {
	var sign = fmt.Sprintf("%s%d%d", r.RemoteAddr, time.Now().Nanosecond(), rand.Intn(100000))
	h := md5.New()
	h.Write([]byte(sign))
	return hex.EncodeToString(h.Sum(nil))
}
