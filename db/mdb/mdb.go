package mdb

import (
	"sync"

	"github.com/wswz/go_commons/conf"
	"github.com/wswz/go_commons/libs/mdb"
)

var (
	dbs    = make(map[string]*mdb.MdbSession)
	mutext sync.Mutex
)

func GetDb(name string) *mdb.MdbSession {
	if _, ok := dbs[name]; !ok {
		mutext.Lock()
		defer mutext.Unlock()
		cfg := conf.GetResConfig().Mongo[name]
		dbs[name] = mdb.GetByRepl(cfg)
		//todo
		// dbs[name].Session().SetMode(mgo.SecondaryPreferred, true)
	}
	return dbs[name]
}
