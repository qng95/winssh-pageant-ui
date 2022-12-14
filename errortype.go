package main

import (
	"errors"
)

// BEGIN: StepCLi Errors Section

var (
	STEPERR_STEPCLI_NOT_FOUND = errors.New("step.exe not found")

	STEPERR_NO_USER_CONFIGURED        = errors.New("no user configured")
	STEPERR_STEPCA_NOT_CONFIGURED     = errors.New("no step ca configured")
	STEPERR_NO_STEP_TEAM_CONFIGURED   = errors.New("no step team configured")
	STEPERR_NO_PROVISIONER_CONFIGURED = errors.New("no provisioner configured")

	STEPERR_LOGIN_FAILED            = errors.New("step user login failed")
	STEPERR_LOGOUT_FAILED           = errors.New("step user logout failed")
	STEPERR_RECONFIGURATION_FAILED  = errors.New("step reconfiguration failed")
	STEPERR_GET_PROVISIONERS_FAILED = errors.New("get step provisioner failed")

	STEPERR_NO_USERCERT_FOUND = errors.New("no user's certificate found in openssh-agent")
	STEPERR_USERCERT_EXPIRED  = errors.New("user's certificate expired")

	STEPERR_INVALID_TEAM_URL            = errors.New("invalid team's url")
	STEPERR_INVALID_USERCERT_VALID_DATE = errors.New("invalid user's certificate validation date")

	STEPERR_CASERVER_ERROR_INTERNAL_SERVER_ERROR = errors.New("CA server has internal error")
	STEPERR_CASERVER_ERROR_SERVICE_UNAVAILABLE   = errors.New("CA server: Service Unavailble")

	STEPERR_UNKNOWN_ERROR = errors.New("unknown step error")

	STEPERR_IDENTITY_NOT_FOUND = errors.New("Identity not found")
)

// END: StepCli Errors Section

// BEGIN: PowerShell Errors Type

var (
	PSERR_PS_EXE_NOTFOUND   = errors.New("powershell.exe not found")
	PSERR_EXECUTION_FAILED  = errors.New("command execution failed")
	PSERR_NON_ZERO_EXIZCODE = errors.New("command execution return non-zero exit code")
)

// END: PowerShell Errors Type
