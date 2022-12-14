package main

import (
	"os"
	"path/filepath"

	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

const (
	APP_NAME = "WinSSH-Pageant-UI"

	CRYPTPROTECTMEMORY_BLOCK_SIZE    = 16
	CRYPTPROTECTMEMORY_CROSS_PROCESS = 1
	FILE_MAP_ALL_ACCESS              = 0xf001f

	// Pageant consts
	AGENT_PIPE_NAME   = `\\.\pipe\pageant.%s.%s`
	AGENT_COPYDATA_ID = 0x804e50ba
	WND_CLASSNAME     = "Pageant"

	// windows ssh-agent pipe name
	SSH_AGENT_PIPE = `\\.\pipe\openssh-ssh-agent`

	SE_KERNAL_OBJECT           = 6
	OWNER_SECURITY_INFORMATION = 1
)

var (
	USER_HOME_DIR, _ = os.UserHomeDir()
	APP_HOME_DIR     = filepath.Join(USER_HOME_DIR, ".winssh_pageantui")
	APP_LOGS_DIR     = filepath.Join(APP_HOME_DIR, "logs")
	APP_CONFS_DIR    = filepath.Join(APP_HOME_DIR, "configs")
	APP_CONFS_FILE   = filepath.Join(APP_CONFS_DIR, "default-conf.json")

	CRYPT_32                  = syscall.NewLazyDLL("crypt32.dll")
	PROC_CRYPT_PROTECT_MEMORY = CRYPT_32.NewProc("CryptProtectMemory")

	MOD_KERNEL32            = syscall.NewLazyDLL("kernel32.dll")
	PROC_OPENFILE_MAPPING_A = MOD_KERNEL32.NewProc("OpenFileMappingA")

	MOD_ADV_API32          = windows.NewLazySystemDLL("advapi32.dll")
	PROC_GET_SECURITY_INFO = MOD_ADV_API32.NewProc("GetSecurityInfo")

	STARTUP_DATE = time.Now().Format("2006-01-02")
)

func initPaths() {
	os.MkdirAll(APP_HOME_DIR, os.ModePerm)
	os.MkdirAll(APP_LOGS_DIR, os.ModePerm)
	os.MkdirAll(APP_CONFS_DIR, os.ModePerm)
}
