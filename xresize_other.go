//go:build !linux || wayland

package main

func SendResizeToPID(pid, w, h int) {}
