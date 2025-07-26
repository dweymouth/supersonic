package util

import (
	"context"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

type FileStreamerServer struct {
	Path       string
	IsComplete func() bool
	listener   net.Listener
	server     *http.Server
	done       chan struct{}
}

// NewFileStreamerServer creates a new server but doesn't start it yet.
func NewFileStreamerServer(path string, isComplete func() bool) (*FileStreamerServer, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return nil, err
	}

	fs := &FileStreamerServer{
		Path:       path,
		IsComplete: isComplete,
		listener:   listener,
		done:       make(chan struct{}),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/stream", fs.streamHandler)

	fs.server = &http.Server{
		Handler: mux,
	}

	return fs, nil
}

// Addr returns the server address (host:port).
func (fs *FileStreamerServer) Addr() string {
	_, port, _ := net.SplitHostPort(fs.listener.Addr().String())
	return "http://localhost:" + port + "/stream"
}

// Serve starts serving and waits for a single request to complete.
func (fs *FileStreamerServer) Serve() error {
	go func() {
		_ = fs.server.Serve(fs.listener)
	}()

	<-fs.done // wait for the handler to finish

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return fs.server.Shutdown(ctx)
}

// Handler that streams the file using chunked transfer encoding.
func (fs *FileStreamerServer) streamHandler(w http.ResponseWriter, r *http.Request) {
	defer close(fs.done) // signal Serve() to shut down after this request

	file, err := os.Open(fs.Path)
	if err != nil {
		http.Error(w, "could not open file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.WriteHeader(http.StatusOK)

	flusher, canFlush := w.(http.Flusher)

	buf := make([]byte, 4096)
	for {
		n, err := file.Read(buf)
		if err != nil && err != io.EOF {
			log.Printf("read error: %v", err)
			break
		}

		if n > 0 {
			_, err := w.Write(buf[:n])
			if err != nil {
				log.Printf("client write error: %v", err)
				break
			}
			if canFlush {
				flusher.Flush()
			}
		}

		if n == 0 && fs.IsComplete() {
			break
		}

		// Wait for more content to be written to the source file
		if n == 0 {
			time.Sleep(50 * time.Millisecond)
		}
	}
}
