package main

import (
	"bytes"
	"os/exec"
	"strings"
	"syscall"
)

type PowerShellType struct {
	powerShellExe string
}

// New create new session
func NewPowershell() (*PowerShellType, error) {
	ps, err := exec.LookPath("cmd.exe")
	//ps, err := exec.LookPath("powershell.exe")
	if err != nil {
		Logger.Panic("Could not found powershell.exe. Error: %v", err)
		return nil, PSERR_PS_EXE_NOTFOUND
	}

	return &PowerShellType{ps}, nil
}

func (p *PowerShellType) ExecuteQuiet(args ...string) (StdOut, StdErr, error) {
	//args = append([]string{"-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden"}, args...)
	args = append([]string{"/C"}, args...)
	cmd := exec.Command(p.powerShellExe, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	cmdErr := cmd.Run()
	stdOut, stdErr := stdout.String(), stderr.String()
	if cmdErr != nil || strings.HasPrefix(stdOut, "ERROR") {
		Logger.Error("Executed command with returned error code '%v'. Error: %v. StdErr: %v", cmd.String(), cmdErr, stdErr)
	}
	return StdOut(stdOut), StdErr(stdErr), cmdErr
}

func (p *PowerShellType) Execute(args ...string) (StdOut, StdErr, error) {
	//args = append([]string{""}, args...)
	cmd := exec.Command(p.powerShellExe, args...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	cmdErr := cmd.Run()
	stdOut, stdErr := stdout.String(), stderr.String()
	if cmdErr != nil || strings.HasPrefix(stdOut, "ERROR") {
		Logger.Error("Execute command with returned error code '%v'. Error: %v. StdErr: %v", cmd.String(), cmdErr, stdErr)
	}
	return StdOut(stdOut), StdErr(stdErr), cmdErr
}
