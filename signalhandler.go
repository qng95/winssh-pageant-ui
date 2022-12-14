package main

import (
	"os"
	"os/signal"

	"fmt"
	"log"
	"syscall"
	//"runtime/debug"
)

type SignalHandlerType struct{}

var (
	SignalHandler *SignalHandlerType = &SignalHandlerType{}
)

func exitHandler() {
	if r := recover(); r != nil {
		message := fmt.Sprintf("%v", r)
		if message == "runtime error: invalid memory address or nil pointer dereference" {
			return
		}
	}
	Configs.StoreConfigs()
	App.CleanUp()
}

func (sh *SignalHandlerType) Init() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	go func() {
		sig := <-sigs
		log.Println(fmt.Sprintf("Receive signal %v. Teminating", sig))
		exitHandler()
		os.Exit(1)
	}()
}
