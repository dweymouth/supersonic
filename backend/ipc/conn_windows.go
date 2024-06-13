//go:build windows

package ipc

import (
	"net"
	"os/user"
	"regexp"

	"github.com/Microsoft/go-winio"
)

var pipeName = `\\.\pipe\supersonic`

func init() {
	if user, err := user.Current(); err == nil {
		pipeName += regexp.MustCompile(`[^a-zA-Z0-9]+`).ReplaceAllString(user.Name, "")
	}
}

func Dial() (net.Conn, error) {
	return winio.DialPipe(pipeName, nil)
}

func Listen() (net.Listener, error) {
	return winio.ListenPipe(pipeName, nil)
}

func DestroyConn() error {
	// Windows named pipes automatically clean up
	return nil
}
