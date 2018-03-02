package idgen

import (
	"github.com/sumory/idgen"
)

var (
	worker *idgen.IdWorker
)

func InitWorker(workerId int64) {
	var err error
	err, worker = idgen.NewIdWorker(workerId)
	if err != nil {
		panic(err)
	}
}

func NextId() (id int64, err error) {
	err, id = worker.NextId()
	return
}
