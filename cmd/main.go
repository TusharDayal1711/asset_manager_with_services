package main

import (
	"asset/server"
	"os"
	"os/signal"
	"syscall"
)

func main() {

	srv := server.ServerInit()
	go srv.Start()
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-done
	srv.Stop()
}
