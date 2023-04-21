package backend

import (
	"log"
	"strconv"

	"github.com/dweymouth/go-subsonic/subsonic"
)

type searchIterBase struct {
	query        string
	artistOffset int
	albumOffset  int
	songOffset   int
	s            *subsonic.Client
}

func (s *searchIterBase) fetchResults() *subsonic.SearchResult3 {
	searchOpts := map[string]string{
		"artistOffset": strconv.Itoa(s.artistOffset),
		"albumOffset":  strconv.Itoa(s.albumOffset),
		"songOffset":   strconv.Itoa(s.songOffset),
	}
	results, err := s.s.Search3(s.query, searchOpts)
	if err != nil {
		log.Println(err)
		results = nil
	}
	if results == nil || len(results.Album)+len(results.Artist)+len(results.Song) == 0 {
		return nil
	}
	return results
}
