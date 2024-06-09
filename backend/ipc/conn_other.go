//go:build !windows

package ipc

import "net"

func Dial() (net.Conn, error) {
	// TODO - use XDG runtime dir, also handle portable mode
	return net.Dial("unix", "/tmp/supersonic.sock")
}

func Listen() (net.Listener, error) {
	return net.Listen("unix", "/tmp/supersonic.sock")
}
