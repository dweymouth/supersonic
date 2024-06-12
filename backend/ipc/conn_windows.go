//go:build windows

package ipc

import (
	"net"
	"time"

	"github.com/Microsoft/go-winio"
)

func Dial() (net.Conn, error) {
	timeout := 300 * time.Millisecond
	return winio.DialPipe("supersonic", &timeout)
}

func Listen() (net.Listener, error) {
	return winio.ListenPipe("supersonic", nil)
}

func DestroyConn() error {
	// Windows named pipes automatically clean up
	return nil
}
