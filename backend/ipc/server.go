package ipc

import (
	"encoding/json"
	"net/http"
)

type Handler interface {
	PlayPause() error
	Stop() error
	Pause() error
	Continue() error
	SeekBackOrPrevious() error
	SeekNext() error
	SeekSeconds(float64) error
	Volume() int
	SetVolume(int) error
}

type serverImpl struct {
	handler Handler
}

func NewServer(handler Handler) *http.Server {
	s := serverImpl{handler: handler}
	return &http.Server{
		Handler: s.createHandler(),
	}
}

func (s *serverImpl) createHandler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("The given path is not valid"))
	})
	m.HandleFunc(PlayPath, s.makeSimpleEndpointHandler(s.handler.Continue))
	m.HandleFunc(PausePath, s.makeSimpleEndpointHandler(s.handler.Pause))
	m.HandleFunc(PlayPausePath, s.makeSimpleEndpointHandler(s.handler.PlayPause))
	m.HandleFunc(StopPath, s.makeSimpleEndpointHandler(s.handler.Stop))
	m.HandleFunc(PreviousPath, s.makeSimpleEndpointHandler(s.handler.SeekBackOrPrevious))
	m.HandleFunc(NextPath, s.makeSimpleEndpointHandler(s.handler.SeekNext))
	return m
}

func (s *serverImpl) makeSimpleEndpointHandler(f func() error) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := f(); err == nil {
			s.writeOK(w)
		} else {
			s.writeErr(w, err)
		}
	}
}

func (s *serverImpl) writeOK(w http.ResponseWriter) (int, error) {
	var r Response
	b, err := json.Marshal(&r)
	if err != nil {
		return 0, err
	}
	return w.Write(b)
}

func (s *serverImpl) writeErr(w http.ResponseWriter, err error) (int, error) {
	r := Response{Error: err.Error()}
	b, err := json.Marshal(&r)
	if err != nil {
		return 0, err
	}
	w.WriteHeader(http.StatusInternalServerError)
	return w.Write(b)
}
