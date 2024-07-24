//go:build windows

package backend

import (
	"sync/atomic"
	"syscall"
)

const (
	ES_CONTINUOUS      uint = 0x80000000
	ES_SYSTEM_REQUIRED uint = 0x00000001
)

var (
	sleepDisabled  atomic.Bool
	executionState *syscall.LazyProc
)

func SetSystemSleepDisabled(disable bool) {
	if old := sleepDisabled.Swap(disable); old == disable {
		return
	}

	if executionState == nil {
		kernel32 := syscall.NewLazyDLL("kernel32.dll")
		executionState = kernel32.NewProc("SetThreadExecutionState")
	}

	uType := ES_CONTINUOUS
	if disable {
		uType |= ES_SYSTEM_REQUIRED
	}

	syscall.SyscallN(executionState.Addr(), 1, uintptr(uType), 0, 0)
}
