//go:build !darwin && !windows

package controller

func setWindowDarkTheme(ptr uintptr, mode int) {}
