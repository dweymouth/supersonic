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

// socketPath is automatically initialized based on platform conventions:
//   - macOS: ~/Library/Caches/supersonic/supersonic.sock (or /tmp/supersonic-{uid}.sock as fallback)
//   - Linux/Unix: $XDG_RUNTIME_DIR/supersonic.sock (or /tmp/supersonic-{uid}.sock as fallback)
//
// TODO: Add support for portable mode by allowing override via environment variable
// or configuration file (e.g., SUPERSONIC_SOCKET_PATH).
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

// Dial establishes a connection to the IPC socket.
// Returns an error if the socket doesn't exist or connection fails.
func Dial() (net.Conn, error) {
	return net.Dial("unix", socketPath)
}

// Listen creates a Unix domain socket listener at the configured path.
// The socket file is created automatically and should be cleaned up
// with DestroyConn() when done.
func Listen() (net.Listener, error) {
	return net.Listen("unix", socketPath)
}

// DestroyConn removes the Unix socket file from the filesystem.
// Should be called during application shutdown.
func DestroyConn() error {
	return os.Remove(socketPath)
}
