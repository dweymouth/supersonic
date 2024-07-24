//go:build !windows

package backend

func SetSystemSleepDisabled(disable bool) {
	// no-op
}
