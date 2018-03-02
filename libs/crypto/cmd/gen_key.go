package main

import (
	"encoding/hex"
	"fmt"

	"github.com/rogpeppe/fastuuid"
)

func main() {
	g, err := fastuuid.NewGenerator()
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	uid := g.Next()
	ekey := hex.EncodeToString(uid[:])

	g, err = fastuuid.NewGenerator()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	uid = g.Next()
	ikey := hex.EncodeToString(uid[:])

	fmt.Printf("ekey:[%s]\nikey:[%s]\n", ekey, ikey)
}
