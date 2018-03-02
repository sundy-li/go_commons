package main

import (
	"log"
	"os"
	// "syscall"
	"time"

	"github.com/wswz/go_commons/app"
)

func main() {
	log.Println("wait 10s with send kill singal")
	go func() {
		time.Sleep(time.Second * 3)
		proc, err := os.FindProcess(os.Getpid())
		if err != nil {
			os.Exit(1)
		}
		proc.Signal(os.Interrupt)
		// return syscall.Kill(os.Getpid(), syscall.SIGINT)
		return
	}()
	app.Run("test app run",
		app.Funcs(
			app.LogWrapper("init", func() error {
				log.Println("init once")
				return nil
			}),
		),
		app.Funcs(
			app.LogWrapper("run", func() error {
				log.Println("run second")
				return nil
			}),
		),
		app.Funcs(
			app.LogWrapper("clean", func() error {
				log.Println("clean third")
				return nil
			}),
		),
	)
}
