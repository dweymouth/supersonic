package dialogs

import (
	// "fmt"
	"fmt"
	"log"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/deluan/sanitize"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/sharedutil"
	"github.com/dweymouth/supersonic/ui/util"
)

type SelectPlaylist struct {
	SearchDialog   *SearchDialog
	mp             mediaprovider.MediaProvider
	loggedInUser   string
	allPlaylists   []*mediaprovider.Playlist
	SkipDuplicates bool
}

func NewSelectPlaylistDialog(mp mediaprovider.MediaProvider, im util.ImageFetcher, loggedInUser string) *SelectPlaylist {

	sp := &SelectPlaylist{
		mp:             mp,
		loggedInUser:   loggedInUser,
		SkipDuplicates: false,
	}
	sd := NewSearchDialog(
		im,
		"Select playlist",
		sp.onSearched,
		sp.onInit,
	)
	sp.SearchDialog = sd
	return sp
}

func (sp *SelectPlaylist) onInit() ([]*mediaprovider.SearchResult, *widget.Check) {
	var results []*mediaprovider.SearchResult
	playlists, err := sp.mp.GetPlaylists()
	if err != nil {
		// TODO: surface this error to user
		log.Printf("error getting playlists: %s", err.Error())
		return results, nil
	}
	sp.allPlaylists = sharedutil.FilterSlice(playlists, func(playlist *mediaprovider.Playlist) bool {
		return playlist.Owner == sp.loggedInUser
	})
	for _, playlist := range sp.allPlaylists {
		results = append(results, &mediaprovider.SearchResult{
			Name:       playlist.Name,
			ID:         playlist.ID,
			CoverID:    playlist.CoverArtID,
			Type:       mediaprovider.ContentTypePlaylist,
			Size:       playlist.TrackCount,
			ArtistName: playlist.Name,
		})
	}
	skipDuplicatesCheck := widget.NewCheck("Skip duplicates", func(checked bool) {
		sp.SkipDuplicates = checked
	})
	return results, skipDuplicatesCheck
}

func (sp *SelectPlaylist) onSearched(query string) []*mediaprovider.SearchResult {
	var results []*mediaprovider.SearchResult
	var filteredPlaylists []*mediaprovider.Playlist
	if query == "" {
		filteredPlaylists = sp.allPlaylists
	} else {
		filteredPlaylists = sharedutil.FilterSlice(sp.allPlaylists, func(playlist *mediaprovider.Playlist) bool {
			return strings.Contains(
				sanitize.Accents(strings.ToLower(playlist.Name)),
				sanitize.Accents(strings.ToLower(query)),
			)
		})
		results = append(results, &mediaprovider.SearchResult{
			Name:  fmt.Sprintf("Create new playlist: %s", query),
			Type:  mediaprovider.ContentTypePlaylist,
			Query: query,
		})
	}

	for _, playlist := range filteredPlaylists {
		results = append(results, &mediaprovider.SearchResult{
			Name:       playlist.Name,
			ID:         playlist.ID,
			CoverID:    playlist.CoverArtID,
			Type:       mediaprovider.ContentTypePlaylist,
			Size:       playlist.TrackCount,
			ArtistName: playlist.Name,
		})
	}
	return results
}

func (sp *SelectPlaylist) SetOnDismiss(onDismiss func()) {
	sp.SearchDialog.OnDismiss = onDismiss
}

func (sp *SelectPlaylist) SetOnNavigateTo(onNavigateTo func(mediaprovider.ContentType, string, string)) {
	sp.SearchDialog.OnNavigateTo = onNavigateTo
}

func (sp *SelectPlaylist) MinSize() fyne.Size {
	return sp.SearchDialog.MinSize()
}

func (sp *SelectPlaylist) GetSearchEntry() fyne.Focusable {
	return sp.SearchDialog.GetSearchEntry()
}
