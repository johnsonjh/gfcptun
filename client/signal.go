// +build linux darwin freebsd

package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	kcp "go.gridfinity.dev/gfcp"
)

func init() {
	go sigHandler()
}

func sigHandler() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGUSR1)
	signal.Ignore(syscall.SIGPIPE)

	for {
		switch <-ch {
		case syscall.SIGUSR1:
			log.Printf("GFCP SNSI:%+v", kcp.DefaultSnsi.Copy())
		}
	}
}
