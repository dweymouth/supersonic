package util

import (
	"net/http/httptest"
	"os"
	"testing"
)

func TestFileStreamerMultipleRequestsSignalDoneOnce(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "stream-*")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("stream data"); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	fs, err := NewFileStreamerServer(f.Name(), func() bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = fs.listener.Close()
	})

	req := httptest.NewRequest("GET", fs.Addr(), nil)
	fs.streamHandler(httptest.NewRecorder(), req)
	fs.streamHandler(httptest.NewRecorder(), req)
}
