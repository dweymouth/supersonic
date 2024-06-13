//go:build !windows

package ipc

import (
	"fmt"
	"net"
	"os"
	"os/user"
	"path"
	"runtime"
)

var socketPath = "/tmp/supersonic.sock"

func init() {
	if runtime.GOOS == "darwin" {
		if home, err := os.UserHomeDir(); err == nil {
			socketPath = path.Join(home, "Library", "Caches", "supersonic", "supersonic.sock")
		} else if user, err := user.Current(); err == nil {
			socketPath = fmt.Sprintf("/tmp/supersonic-%s.sock", user.Uid)
		}
	} else {
		if runtime := os.Getenv("XDG_RUNTIME_DIR"); runtime != "" {
			socketPath = path.Join(runtime, "supersonic.sock")
		} else if user, err := user.Current(); err == nil {
			socketPath = fmt.Sprintf("/tmp/supersonic-%s.sock", user.Uid)
		}
	}
}

func Dial() (net.Conn, error) {
	// TODO - use XDG runtime dir, also handle portable mode
	return net.Dial("unix", socketPath)
}

func Listen() (net.Listener, error) {
	return net.Listen("unix", socketPath)
}

func DestroyConn() error {
	return os.Remove(socketPath)
}
