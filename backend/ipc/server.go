package ipc

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strconv"
)

type PlaybackHandler interface {
	PlayPause()
	Stop()
	Pause()
	Continue()
	SeekBackOrPrevious()
	SeekNext()
	SeekSeconds(float64)
	SeekBySeconds(float64)
	Volume() int
	SetVolume(int)
}

type IPCServer interface {
	Serve(net.Listener) error
	Shutdown(context.Context) error
}

type serverImpl struct {
	server    *http.Server
	pbHandler PlaybackHandler
	showFn    func()
	quitFn    func()
}

func NewServer(pbHandler PlaybackHandler, showFn, quitFn func()) IPCServer {
	s := &serverImpl{pbHandler: pbHandler, showFn: showFn, quitFn: quitFn}
	s.server = &http.Server{
		Handler: s.createHandler(),
	}
	return s
}

func (s *serverImpl) Serve(listener net.Listener) error {
	return s.server.Serve(listener)
}

func (s *serverImpl) Shutdown(ctx context.Context) error {
	err := s.server.Shutdown(ctx)
	DestroyConn()
	return err
}

func (s *serverImpl) createHandler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("The given path is not valid"))
	})
	m.HandleFunc(PingPath, s.makeSimpleEndpointHandler(func() {}))
	m.HandleFunc(ShowPath, s.makeSimpleEndpointHandler(func() {
		s.showFn()
	}))
	m.HandleFunc(QuitPath, s.makeSimpleEndpointHandler(func() {
		s.quitFn()
	}))
	m.HandleFunc(PlayPath, s.makeSimpleEndpointHandler(s.pbHandler.Continue))
	m.HandleFunc(PausePath, s.makeSimpleEndpointHandler(s.pbHandler.Pause))
	m.HandleFunc(PlayPausePath, s.makeSimpleEndpointHandler(s.pbHandler.PlayPause))
	m.HandleFunc(StopPath, s.makeSimpleEndpointHandler(s.pbHandler.Stop))
	m.HandleFunc(PreviousPath, s.makeSimpleEndpointHandler(s.pbHandler.SeekBackOrPrevious))
	m.HandleFunc(NextPath, s.makeSimpleEndpointHandler(s.pbHandler.SeekNext))
	m.HandleFunc(TimePosPath, s.makeFloatEndpointHandler(s.pbHandler.SeekSeconds, "s"))
	m.HandleFunc(SeekByPath, s.makeFloatEndpointHandler(s.pbHandler.SeekBySeconds, "s"))
	m.HandleFunc(VolumePath, func(w http.ResponseWriter, r *http.Request) {
		v := r.URL.Query().Get("v")
		if vol, err := strconv.Atoi(v); err == nil {
			s.pbHandler.SetVolume(vol)
			s.writeOK(w)
		} else {
			s.writeErr(w, err)
		}
	})
	return m
}

func (s *serverImpl) makeSimpleEndpointHandler(f func()) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		f()
		s.writeOK(w)
	}
}

func (s *serverImpl) makeFloatEndpointHandler(f func(float64), queryParam string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		v := r.URL.Query().Get(queryParam)
		if val, err := strconv.ParseFloat(v, 64); err == nil {
			f(val)
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
