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
	m.HandleFunc(PingPath, s.makeSimpleEndpointHandler(func() error { return nil }))
	m.HandleFunc(PlayPath, s.makeSimpleEndpointHandler(s.handler.Continue))
	m.HandleFunc(PausePath, s.makeSimpleEndpointHandler(s.handler.Pause))
	m.HandleFunc(PlayPausePath, s.makeSimpleEndpointHandler(s.handler.PlayPause))
	m.HandleFunc(StopPath, s.makeSimpleEndpointHandler(s.handler.Stop))
	m.HandleFunc(PreviousPath, s.makeSimpleEndpointHandler(s.handler.SeekBackOrPrevious))
	m.HandleFunc(NextPath, s.makeSimpleEndpointHandler(s.handler.SeekNext))
	m.HandleFunc(TimePosPath, func(w http.ResponseWriter, r *http.Request) {
		var t TimePos
		if err := json.NewDecoder(r.Response.Body).Decode(&t); err != nil {
			s.writeErr(w, err)
			return
		}
		s.writeSimpleResponse(w, s.handler.SeekSeconds(t.Seconds))
	})
	m.HandleFunc(VolumePath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			msg, _ := json.Marshal(Volume{Volume: s.handler.Volume()})
			w.Write(msg)
			return
		}
		var v Volume
		if err := json.NewDecoder(r.Response.Body).Decode(&v); err != nil {
			s.writeErr(w, err)
			return
		}
		s.writeSimpleResponse(w, s.handler.SetVolume(v.Volume))
	})
	return m
}

func (s *serverImpl) makeSimpleEndpointHandler(f func() error) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		s.writeSimpleResponse(w, f())
	}
}

func (s *serverImpl) writeSimpleResponse(w http.ResponseWriter, err error) {
	if err == nil {
		s.writeOK(w)
	} else {
		s.writeErr(w, err)
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
