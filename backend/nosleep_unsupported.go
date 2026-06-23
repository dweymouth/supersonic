//go:build !windows && !darwin

package backend

func SetSystemSleepDisabled(disable bool) {
	// no-op
}
