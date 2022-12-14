package main

import (
	"fmt"
	"log"
)

func main() {
	defer exitHandler()

	initPaths()
	SignalHandler.Init()
	Logger.Init()

	log.Println(fmt.Sprintf(`
	#########################################
		%v STARTING UP %v
	############################################
	`, STARTUP_DATE, APP_NAME))

	Logger.Info("Load/Create application configs")
	Configs.CheckAndCreateConfigFile()
	Configs.LoadConfigs()
	
	App.Init()
	App.Start()
}
