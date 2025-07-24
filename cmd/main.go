package main

import (
	"asset/server"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	srv := server.SrvInit()
	go srv.Start()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	
	<-quit
	srv.Stop()
}
