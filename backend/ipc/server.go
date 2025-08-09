package ipc

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/dweymouth/supersonic/backend/mediaprovider"
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
	PlayAlbum(string, int, bool) error
	PlayPlaylist(string, int, bool) error
	PlayTrack(string) error
}

type IPCServer interface {
	Serve(net.Listener) error
	Shutdown(context.Context) error
}

type serverImpl struct {
	server    *http.Server
	pbHandler PlaybackHandler
	mp        *mediaprovider.MediaProvider
	showFn    func()
	quitFn    func()
}

func NewServer(pbHandler PlaybackHandler, mp *mediaprovider.MediaProvider, showFn, quitFn func()) IPCServer {
	s := &serverImpl{pbHandler: pbHandler, mp: mp, showFn: showFn, quitFn: quitFn}
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
	m.HandleFunc(TimePosPath, s.makeFloatEndpointHandler("s", s.pbHandler.SeekSeconds))
	m.HandleFunc(SeekByPath, s.makeFloatEndpointHandler("s", s.pbHandler.SeekBySeconds))
	m.HandleFunc(VolumePath, func(w http.ResponseWriter, r *http.Request) {
		v := r.URL.Query().Get("v")
		if vol, err := strconv.Atoi(v); err == nil {
			s.pbHandler.SetVolume(vol)
			s.writeOK(w)
		} else {
			s.writeErr(w, err)
		}
	})
	m.HandleFunc(VolumeAdjustPath, s.makeFloatEndpointHandler("pct", func(pct float64) {
		vol := s.pbHandler.Volume()
		vol = vol + int(float64(vol)*(pct/100))
		s.pbHandler.SetVolume(vol) // will clamp to range for us
	}))
	m.HandleFunc(PlayAlbumPath, s.makeTracklistEndpointHandler(s.pbHandler.PlayAlbum))
	m.HandleFunc(PlayPlaylistPath, s.makeTracklistEndpointHandler(s.pbHandler.PlayPlaylist))
	m.HandleFunc(PlayTrackPath, func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		s.pbHandler.PlayTrack(id)
		s.writeOK(w)
	})
	m.HandleFunc(SearchAlbumPath, s.makeSearchEndpointHandler(func(search string) (any, error) {
		filter := mediaprovider.NewAlbumFilter(mediaprovider.AlbumFilterOptions{})
		i := (*s.mp).SearchAlbums(search, filter)

		album := i.Next()
		albums := make([]mediaprovider.Album, 0)
		for album != nil {
			albums = append(albums, *album)
			album = i.Next()
		}

		return albums, nil
	}))
	m.HandleFunc(SearchPlaylistPath, s.makeSearchEndpointHandler(func(search string) (any, error) {
		all, err := (*s.mp).GetPlaylists()
		if err != nil {
			return nil, err
		}

		search = strings.ReplaceAll(search, " ", "")
		search = strings.ToLower(search)

		filtered := make([]mediaprovider.Playlist, 0)
		for i := 0; i < len(all); i++ {
			playlist := all[i]
			name := strings.ReplaceAll(playlist.Name, " ", "")
			name = strings.ToLower(name)
			if strings.Contains(name, search) {
				filtered = append(filtered, *playlist)
			}
		}

		return filtered, nil
	}))
	m.HandleFunc(SearchTrackPath, s.makeSearchEndpointHandler(func(search string) (any, error) {
		i := (*s.mp).IterateTracks(search)

		track := i.Next()
		tracks := make([]mediaprovider.Track, 0)
		for track != nil {
			tracks = append(tracks, *track)
			track = i.Next()
		}

		return tracks, nil
	}))
	return m
}

func (s *serverImpl) makeSimpleEndpointHandler(f func()) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		f()
		s.writeOK(w)
	}
}

func (s *serverImpl) makeFloatEndpointHandler(queryParam string, f func(float64)) func(http.ResponseWriter, *http.Request) {
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

func (s *serverImpl) makeTracklistEndpointHandler(f func(string, int, bool) error) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		id := query.Get("id")
		firstTrack, err := strconv.Atoi(query.Get("t"))
		if err != nil {
			s.writeErr(w, err)
			return
		}
		shuffle, err := strconv.ParseBool(query.Get("s"))
		if err != nil {
			s.writeErr(w, err)
			return
		}
		f(id, firstTrack, shuffle)
		s.writeOK(w)
	}
}

func (s *serverImpl) makeSearchEndpointHandler(f func(string) (any, error)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		search := r.URL.Query().Get("s")
		data, err := f(search)
		if err != nil {
			s.writeErr(w, err)
			return
		}
		bytes, err := json.Marshal(data)
		if err != nil {
			s.writeErr(w, err)
		}
		s.writeData(w, bytes)
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

func (s *serverImpl) writeData(w http.ResponseWriter, data []byte) (int, error) {
	r := Response{Data: data}
	b, err := json.Marshal(&r)
	if err != nil {
		return 0, err
	}
	w.Header().Set("Content-Type", "application/json")
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
