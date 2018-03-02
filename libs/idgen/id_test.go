package idgen

import "testing"

func TestNextId(t *testing.T) {
	InitWorker(1)

	aa, err := NextId()
	if err != nil {
		panic(err)
	}
	println(aa)
}
