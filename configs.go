package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

type ConfigType struct {
	StepTeamUrl            string
	StepTeamName           string
	StepDefaultProvisioner string
	StepUsername           string
}

var (
	Configs = &ConfigType{
		StepTeamUrl:            "",
		StepTeamName:           "",
		StepDefaultProvisioner: "",
		StepUsername:           "",
	}
)

func (c *ConfigType) CheckAndCreateConfigFile() {
	if _, err := os.Stat(APP_CONFS_FILE); os.IsNotExist(err) {
		Logger.Info("Config file not exist at %v. Will created default one.", APP_CONFS_FILE)
		Configs.StoreConfigs() // create config file with empty content
	}
}

func (currentConfigs *ConfigType) LoadConfigs() {
	content, err := ioutil.ReadFile(APP_CONFS_FILE)
	if err != nil {
		Logger.Error("Failed to read configs from %v with errors: %v", APP_CONFS_FILE, err)
		return
	}

	currentConfig := ConfigType{}
	err = json.Unmarshal([]byte(content), &currentConfig)
	if err != nil {
		Logger.Error("Failed to load unmarshal configs in  %v with errors: %v", APP_CONFS_FILE, err)
		return
	}

	Configs.UpdateConfig(currentConfig)
	Logger.Info("Loaded config from %v", APP_CONFS_FILE)
}

func (currentConfigs *ConfigType) StoreConfigs() {
	content, err := json.MarshalIndent(currentConfigs, "", " ")
	if err != nil {
		Logger.Error("Failed to marshal current configs with errors: %v", err)
		return
	}

	err = ioutil.WriteFile(APP_CONFS_FILE, content, 0664)
	if err != nil {
		Logger.Error("Failed to store configs to %v with errors: %v", APP_CONFS_FILE, err)
		return
	}

	Logger.Info("Stored config in %v", APP_CONFS_FILE)
}

func (currentConfig *ConfigType) UpdateConfig(newConfig ConfigType) {
	if newConfig.StepTeamUrl != "" {
		Logger.Info("Updating new step team url into configs")
		currentConfig.StepTeamUrl = newConfig.StepTeamUrl
	}

	if newConfig.StepUsername != "" {
		Logger.Info("Updating new step username '%v' into configs", newConfig.StepUsername)
		currentConfig.StepUsername = newConfig.StepUsername
	}

	if newConfig.StepDefaultProvisioner != "" {
		Logger.Info("Updating new step ca provisioner '%v' into configs", newConfig.StepDefaultProvisioner)
		currentConfig.StepDefaultProvisioner = newConfig.StepDefaultProvisioner
	}

	if newConfig.StepTeamName != "" {
		Logger.Info("Updating new step team '%v' into configs", newConfig.StepTeamName)
		currentConfig.StepTeamName = newConfig.StepTeamName
	}
}
