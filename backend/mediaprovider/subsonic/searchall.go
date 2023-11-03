package subsonic

import (
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/deluan/sanitize"
	"github.com/dweymouth/go-subsonic/subsonic"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/sharedutil"
)

func (s *subsonicMediaProvider) SearchAll(searchQuery string, maxResults int) ([]*mediaprovider.SearchResult, error) {
	var wg sync.WaitGroup
	var err error // only set by Search3
	var result *subsonic.SearchResult3
	var playlists []*subsonic.Playlist
	var genres []*subsonic.Genre

	wg.Add(1)
	go func() {
		count := strconv.Itoa(maxResults / 3)
		res, e := s.client.Search3(searchQuery, map[string]string{
			"artistCount": count,
			"albumCount":  count,
			"songCount":   count,
		})
		if e != nil {
			err = e
		} else {
			result = res
		}
		wg.Done()
	}()

	queryLowerWords := strings.Fields(strings.ToLower(sanitize.Accents(searchQuery)))

	wg.Add(1)
	go func() {
		p, e := s.client.GetPlaylists(nil)
		if e == nil {
			playlists = sharedutil.FilterSlice(p, func(p *subsonic.Playlist) bool {
				return allTermsMatch(strings.ToLower(sanitize.Accents(p.Name)), queryLowerWords)
			})
		}
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		g, e := s.client.GetGenres()
		if e == nil {
			genres = sharedutil.FilterSlice(g, func(g *subsonic.Genre) bool {
				return allTermsMatch(strings.ToLower(sanitize.Accents(g.Name)), queryLowerWords)
			})
		}
		wg.Done()
	}()

	wg.Wait()
	if err != nil {
		return nil, err
	}

	results := mergeResults(result, playlists, genres)
	rankResults(results, queryLowerWords)
	if len(results) > maxResults {
		results = results[:maxResults]
	}
	return results, nil
}

// name and terms should be pre-converted to the same case
func allTermsMatch(name string, terms []string) bool {
	for _, t := range terms {
		if !strings.Contains(name, t) {
			return false
		}
	}
	return true
}

func mergeResults(
	searchResult *subsonic.SearchResult3,
	matchingPlaylists []*subsonic.Playlist,
	matchingGenres []*subsonic.Genre,
) []*mediaprovider.SearchResult {
	var results []*mediaprovider.SearchResult

	for _, al := range searchResult.Album {
		results = append(results, &mediaprovider.SearchResult{
			Type:       mediaprovider.ContentTypeAlbum,
			ID:         al.ID,
			CoverID:    al.CoverArt,
			Name:       al.Name,
			ArtistName: getNameString(al.Artist, al.Artists),
			Size:       al.SongCount,
		})
	}

	for _, ar := range searchResult.Artist {
		results = append(results, &mediaprovider.SearchResult{
			Type:    mediaprovider.ContentTypeArtist,
			ID:      ar.ID,
			CoverID: ar.CoverArt,
			Name:    ar.Name,
			Size:    ar.AlbumCount,
		})
	}

	for _, tr := range searchResult.Song {
		results = append(results, &mediaprovider.SearchResult{
			Type:       mediaprovider.ContentTypeTrack,
			ID:         tr.ID,
			CoverID:    tr.CoverArt,
			Name:       tr.Title,
			ArtistName: getNameString(tr.Artist, tr.Artists),
			Size:       tr.Duration,
		})
	}

	for _, pl := range matchingPlaylists {
		results = append(results, &mediaprovider.SearchResult{
			Type:    mediaprovider.ContentTypePlaylist,
			ID:      pl.ID,
			CoverID: pl.CoverArt,
			Name:    pl.Name,
			Size:    pl.SongCount,
		})
	}

	for _, g := range matchingGenres {
		results = append(results, &mediaprovider.SearchResult{
			Type: mediaprovider.ContentTypeGenre,
			ID:   g.Name,
			Name: g.Name,
			Size: g.AlbumCount,
		})
	}

	return results
}

func rankResults(results []*mediaprovider.SearchResult, queryTerms []string) {
	if len(queryTerms) == 0 || len(results) < 2 {
		return
	}

	sanitizeMemo := make([]string, len(results))
	sanitized := func(s string, i int) string {
		if x := sanitizeMemo[i]; x != "" {
			return x
		}
		x := strings.ToLower(sanitize.Accents(s))
		sanitizeMemo[i] = x
		return x
	}

	sort.Slice(results, func(i, j int) bool {
		// Compare by search query terms
		a, b := results[i], results[j]
		aName := sanitized(a.Name, i)
		bName := sanitized(b.Name, j)

		for _, term := range queryTerms {
			firstTermIdxA, firstTermIdxB := strings.Index(aName, term), strings.Index(bName, term)
			if firstTermIdxA >= 0 && firstTermIdxB < 0 {
				return true // item A has a direct match with the query term and B does not
			} else if firstTermIdxB >= 0 && firstTermIdxA < 0 {
				return false // item B matches but not A
			}

			if firstTermIdxA < firstTermIdxB {
				return true // item A matches the query term starting at an earlier position
			} else if firstTermIdxB < firstTermIdxA {
				return false // item B matches first
			}
		}
		// Defer to item type for priority order
		return a.Type < b.Type
	})
}

// select Subsonic single-valued name or join OpenSubsonic multi-valued names
func getNameString(singleName string, idNames []subsonic.IDName) string {
	if len(idNames) == 0 {
		return singleName
	}
	names := sharedutil.MapSlice(idNames, func(a subsonic.IDName) string {
		return a.Name
	})
	return strings.Join(names, ", ")
}
