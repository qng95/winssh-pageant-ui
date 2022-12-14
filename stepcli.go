package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

var PRINCIPALS []string

func anyStdContains(stdOut StdOut, stdErr StdErr, str string) bool {
	return strings.Contains(string(stdOut), str) || strings.Contains(string(stdErr), str)
}

func parseStepCliError(stdOut StdOut, stdErr StdErr, err error) error {
	if anyStdContains(stdOut, stdErr, "requires the '--ca-url' flag") {
		return STEPERR_STEPCA_NOT_CONFIGURED
	}

	if anyStdContains(stdOut, stdErr, "no key found") {
		return STEPERR_NO_USERCERT_FOUND
	}

	if anyStdContains(stdOut, stdErr, "Internal Server Error") {
		return STEPERR_CASERVER_ERROR_INTERNAL_SERVER_ERROR
	}

	if anyStdContains(stdOut, stdErr, "Service Unavailable") {
		return STEPERR_CASERVER_ERROR_SERVICE_UNAVAILABLE
	}

	if anyStdContains(stdOut, stdErr, "Identity not found") {
		return STEPERR_IDENTITY_NOT_FOUND
	}

	if stdErr != "" || err != nil {
		var errorStr string
		errorStr = string(stdErr)
		if err != nil {
			errorStr = errorStr + " | " + err.Error()
		}
		return errors.New(errorStr)
	}

	return nil
}

const STEP_VALID_CA_TYPE = "OIDC" // only allow OIDC ca type

type StepType struct {
	stepExePath     string
	ps              *PowerShellType
	provisionersSet *HashSet
}

var (
	StepCli *StepType = &StepType{}
)

func (stepcli *StepType) Init() error {
	var err error
	//path, err := exec.LookPath("step.exe")
	if err != nil {
		Logger.Error("Could not found %v . Error: %v", "step.exe", err)
		return STEPERR_STEPCLI_NOT_FOUND
	}

	//stepcli.stepExePath = escapePowershellPath(path)
	stepcli.stepExePath = "step.exe"
	var psErr error
	stepcli.ps, psErr = NewPowershell()
	if psErr != nil {
		Logger.Error("Failed to iniailize StepCLi util. Error: %v", psErr)
		return psErr
	}

	stepcli.provisionersSet = NewHashSet()

	return nil
}

func (stepcli *StepType) Login() error {
	stepUserName := Configs.StepUsername
	if stepUserName == "" {
		return STEPERR_NO_USER_CONFIGURED
	}

	currentProvisioner := Configs.StepDefaultProvisioner
	if currentProvisioner == "" {
		return STEPERR_NO_PROVISIONER_CONFIGURED
	}

	cmdStr := fmt.Sprintf(`%s ssh login %s --provisioner=%s`, stepcli.stepExePath, stepUserName, currentProvisioner)
	Logger.Info("Invoking StepCli.Login Executing: " + cmdStr)
	stdOut, stdErr, psErr := stepcli.ps.ExecuteQuiet(cmdStr)

	stepErr := parseStepCliError(stdOut, stdErr, psErr)
	return stepErr
}

func (stepcli *StepType) Logout() error {
	stepUserName := Configs.StepUsername
	if stepUserName == "" {
		return STEPERR_NO_USER_CONFIGURED
	}

	for _, principal := range PRINCIPALS {
		cmdStr := fmt.Sprintf(`%s ssh logout %s`, stepcli.stepExePath, principal)
		Logger.Info("Invoking StepCli.Logout() Executing: " + cmdStr)
		stdOut, stdErr, psErr := stepcli.ps.ExecuteQuiet(cmdStr)

		stepErr := parseStepCliError(stdOut, stdErr, psErr)
		if stepErr == nil {
			return nil
		}
	}
	return STEPERR_LOGOUT_FAILED
}

func (stepcli *StepType) ReConfigure() error {
	teamName := Configs.StepTeamName
	teamUrl := Configs.StepTeamUrl
	if teamName == "" && teamUrl == "" {
		return STEPERR_NO_STEP_TEAM_CONFIGURED
	}

	var cmdStr string
	if teamName != "" && teamUrl != "" {
		cmdStr = fmt.Sprintf(`%s ssh config --force --team="." --team-url=%s`, stepcli.stepExePath, strings.ReplaceAll(teamUrl, "<>", teamName))
	} else if teamName != "" {
		cmdStr = fmt.Sprintf(`%s ssh config --force --team=%s`, stepcli.stepExePath, teamName)
	} else if teamUrl != "" {
		_, err := url.ParseRequestURI(teamUrl)
		if err != nil {
			Logger.Error("invalid team url %v. Error: %v", err, teamUrl)
			return STEPERR_INVALID_TEAM_URL
		} else {
			cmdStr = fmt.Sprintf(`%s ssh config --force --team-url=%s --team="."`, stepcli.stepExePath, teamUrl)
		}
	} else {
		return STEPERR_NO_STEP_TEAM_CONFIGURED
	}

	Logger.Info("Invoking StepCli.ReConfigure Executing: " + cmdStr)
	cmdFields := strings.Fields(cmdStr)
	stdOut, stdErr, psErr := stepcli.ps.ExecuteQuiet(cmdFields...)

	stepErr := parseStepCliError(stdOut, stdErr, psErr)
	return stepErr
}

func (stepcli *StepType) GetProvisionersSetNoRefresh() (*HashSet, error) {
	return stepcli.GetProvisionersSet(false)
}

func (stepcli *StepType) GetProvisionersSetWithRefreshing() (*HashSet, error) {
	return stepcli.GetProvisionersSet(true)
}

func (stepcli *StepType) GetProvisionersSet(refresh bool) (*HashSet, error) {
	if !refresh {
		return stepcli.provisionersSet, nil
	}

	cmdStr := fmt.Sprintf(`%s ca provisioner list`, stepcli.stepExePath)
	Logger.Info("Invoking StepCli.GetProvisionersSet with refreshing. Executing: " + cmdStr)
	stdOut, stdErr, psErr := stepcli.ps.ExecuteQuiet(cmdStr)
	stepErr := parseStepCliError(stdOut, stdErr, psErr)
	if stepErr != nil {
		return stepcli.provisionersSet, stepErr
	}

	var provisionerList []map[string]interface{}
	json.Unmarshal([]byte(stdOut), &provisionerList)
	for _, provisioner := range provisionerList {
		name := fmt.Sprintf("%v", provisioner["name"])
		ptype := fmt.Sprintf("%v", provisioner["type"])
		if ptype == STEP_VALID_CA_TYPE {
			stepcli.provisionersSet.Add(name)
		}
	}
	return stepcli.provisionersSet, nil
}

func (stepcli *StepType) GetCaHealth() (bool, error) {
	cmdStr := fmt.Sprintf(`%s ca health`, stepcli.stepExePath)
	Logger.Info("Invoking StepCli.GetCaHealth. Executing: " + cmdStr)
	stdOut, stdErr, psErr := stepcli.ps.ExecuteQuiet(cmdStr)
	stepErr := parseStepCliError(stdOut, stdErr, psErr)
	if stepErr != nil {
		return false, stepErr
	}

	caHealthOk := strings.HasPrefix(string(stdOut), "ok")
	if caHealthOk {
		return caHealthOk, nil
	} else {
		return false, STEPERR_STEPCA_NOT_CONFIGURED
	}
}

func (stepcli *StepType) GetUserCertOk() (bool, error) {
	username := Configs.StepUsername
	if username == "" {
		return false, STEPERR_NO_USER_CONFIGURED
	}

	// Get lists of user certs
	cmdStr := fmt.Sprintf("step ssh list --raw")
	Logger.Info("Invoking StepCli.GetUserCertOk. Executing: " + cmdStr)
	stdOut, stdErr, psErr := stepcli.ps.ExecuteQuiet(cmdStr)
	stepErr := parseStepCliError(stdOut, stdErr, psErr)
	if stepErr != nil {
		return false, stepErr
	}

	certs := strings.Split(strings.ReplaceAll(string(stdOut), "\r\n", "\n"), "\n")
	foundUserCert := false
	for _, cert := range certs {
		if cert == "" {
			continue
		}

		cmdStr = fmt.Sprintf(`echo %s | %s ssh inspect --format json`, cert, stepcli.stepExePath)
		Logger.Info("Invoking StepCli.GetUserCertOk. Executing: " + cmdStr)
		stdOut, stdErr, psErr = stepcli.ps.ExecuteQuiet(cmdStr)
		stepErr = parseStepCliError(stdOut, stdErr, psErr)
		if stepErr != nil {
			if strings.Contains(cert, "imported-openssh-key") {
				Logger.Info("Skipping imported-openssh-key: %v", cert)
			} else if strings.HasPrefix(cert, "ssh-rsa") {
				Logger.Info("Skipping RSA key: %v", cert)
			} else {
				Logger.Error("Error while checking certificate: %v", cert)
			}
			continue
		}

		var certDetailMap map[string]interface{}
		json.Unmarshal([]byte(stdOut), &certDetailMap)
		principals := certDetailMap["Principals"]
		for _, p := range principals.([]interface{}) {
			principal := p.(string)
			if principal == username {
				foundUserCert = true

				for _, _p := range principals.([]interface{}) {
					pp := _p.(string)
					PRINCIPALS = append(PRINCIPALS, pp)
				}

				validAfterDateStr := fmt.Sprintf("%v", certDetailMap["ValidAfter"])
				validBeforeDateStr := fmt.Sprintf("%v", certDetailMap["ValidBefore"])

				//2021-11-02T09:56:30+01:00
				validAfterDate, err := time.Parse(time.RFC3339, validAfterDateStr)
				if err != nil {
					return false, STEPERR_INVALID_USERCERT_VALID_DATE
				}
				validBeforeDate, err := time.Parse(time.RFC3339, validBeforeDateStr)
				if err != nil {
					return false, STEPERR_INVALID_USERCERT_VALID_DATE
				}
				vad := validAfterDate.UTC().Unix()
				now := time.Now().UTC().Unix()
				vbd := validBeforeDate.UTC().Unix()

				valid := (vad <= now) && (now <= vbd)
				if valid {
					return valid, nil
				}
			}
		}
	}

	if foundUserCert {
		return false, STEPERR_USERCERT_EXPIRED
	} else {
		return false, STEPERR_NO_USERCERT_FOUND
	}

}
