package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"unsafe"
)

func escapePowershellPath(s string) string {
	//return strings.Replace(s, " ", "` ", -1)
	var ret string
	ret = strings.Replace(s, " ", "^ ", -1)
	ret = strings.Replace(ret, "@", "^@", -1)
	return ret
}

func GetSystemRootPath() string {
	rootPath, ok := os.LookupEnv("SystemRoot")
	if !ok {
		panic(errors.New("could not obtain SystemRoot environment variable"))
	}
	return rootPath
}

func GetUserEmail() string {
	cmd := exec.Command(fmt.Sprintf("%s\\system32\\whoami.exe", GetSystemRootPath()), "/UPN")
	cmdOutput, err := cmd.CombinedOutput()
	if err != nil {
		panic(errors.New("failed to determine user email"))
	}
	userEmail := strings.TrimSpace(strings.TrimSuffix(string(cmdOutput), "\r\n"))
	return userEmail
}

func IsFileExist(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func CapiObfuscateString(realname string) string {
	cryptlen := len(realname) + 1
	cryptlen += CRYPTPROTECTMEMORY_BLOCK_SIZE - 1
	cryptlen /= CRYPTPROTECTMEMORY_BLOCK_SIZE
	cryptlen *= CRYPTPROTECTMEMORY_BLOCK_SIZE

	cryptdata := make([]byte, cryptlen)
	copy(cryptdata, realname)

	pDataIn := uintptr(unsafe.Pointer(&cryptdata[0]))
	cbDataIn := uintptr(cryptlen)
	dwFlags := uintptr(CRYPTPROTECTMEMORY_CROSS_PROCESS)
	// pageant ignores errors
	PROC_CRYPT_PROTECT_MEMORY.Call(pDataIn, cbDataIn, dwFlags)

	hash := sha256.Sum256(cryptdata)
	return hex.EncodeToString(hash[:])
}

func StopProcessWithName(name string) bool {
	ps, psErr := NewPowershell()
	if psErr != nil {
		Logger.Error("Failed to stop process.\nError: %v", name, psErr)
		return false
	}
	//cmdStr := fmt.Sprintf(`Stop-Process -Name "%s" -Force`, name)
	cmdStr := fmt.Sprintf(`taskkill /f /IM %s.exe /T`, name)
	stdOut, stdErr, psErr := ps.ExecuteQuiet(cmdStr)
	if strings.Contains(string(stdOut), "not found") || strings.Contains(string(stdErr), "not found") {
		psErr = nil
	}

	if psErr != nil {
		Logger.Error("Failed to stop process '%v'.\nStdOut: %v.\nStdErr: %v.\nError: %v", name, stdOut, stdErr, psErr)
	}
	return (psErr == nil)
}

func IsProcessNameExist(name string, currentUser bool) bool {
	PIDString := GetProcessPidByName(name, currentUser)
	_, err := strconv.Atoi(PIDString)
	return err == nil
}

func GetProcessPidByName(name string, currentUser bool) string {
	ps, psErr := NewPowershell()
	if psErr != nil {
		Logger.Error("Failed to get PID of process '%v'", psErr)
	}

	//cmdStr := fmt.Sprintf(`Get-Process -Name "%s" -ErrorAction SilentlyContinue | Select -expand Id`, name)
	//cmdStr := fmt.Sprintf("tasklist /fo csv /nh /fi \"USERNAME eq %%username%%\" /fi \"IMAGENAME eq %s.exe\"", name)
	filterProcessName := fmt.Sprintf("IMAGENAME eq %s.exe", name)
	var stdOut StdOut
	if currentUser {
		stdOut, _, psErr = ps.ExecuteQuiet("tasklist", "/fo", "csv", "/nh", "/fi", "USERNAME eq %username%", "/fi", filterProcessName, "|", "findstr", "-i", name+".exe")
	} else {
		stdOut, _, psErr = ps.ExecuteQuiet("tasklist", "/fo", "csv", "/nh", "/fi", filterProcessName, "|", "findstr", "-i", name+".exe")
	}
	if psErr != nil {
		return ""
	}
	stdOurStr := strings.ReplaceAll(string(stdOut), "\"", "")
	tokens := strings.Split(stdOurStr, ",")
	return strings.TrimSpace(tokens[1])
}

func OpenHomeDir() error {
	ps, psErr := NewPowershell()
	if psErr != nil {
		Logger.Error("Failed to open home dir %v", psErr)
		return psErr
	}

	cmdStr := fmt.Sprintf(`explorer %s`, APP_HOME_DIR)
	_, _, psErr = ps.ExecuteQuiet(cmdStr)
	if psErr != nil {
		Logger.Error("Failed to open home dir %v", psErr)
	}
	return psErr
}
