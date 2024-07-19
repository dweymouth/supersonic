//go:build windows

package backend

import "syscall"

const (
	ES_CONTINUOUS      uint = 0x80000000
	ES_SYSTEM_REQUIRED uint = 0x00000001
)

func SetSystemSleepDisabled(disable bool) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	executionState := kernel32.NewProc("SetThreadExecutionState")

	uType := ES_CONTINUOUS
	if disable {
		uType |= ES_SYSTEM_REQUIRED
	}

	syscall.SyscallN(executionState.Addr(), 1, uintptr(uType), 0, 0)
}
