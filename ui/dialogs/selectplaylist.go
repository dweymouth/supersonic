package dialogs

import (
	// "fmt"
	"fmt"
	"log"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/deluan/sanitize"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/sharedutil"
	"github.com/dweymouth/supersonic/ui/util"
)

type SelectPlaylist struct {
	SearchDialog *SearchDialog
	mp           mediaprovider.MediaProvider
	loggedInUser string
}

func NewSelectPlaylistDialog(mp mediaprovider.MediaProvider, im util.ImageFetcher, loggedInUser string) *SelectPlaylist {

	sp := &SelectPlaylist{
		mp:           mp,
		loggedInUser: loggedInUser,
	}

	sd := NewSearchDialog(
		im,
		"Select playlist",
		sp.onSearched,
		sp.onUpdateSearchResult,
	)
	sp.SearchDialog = sd
	return sp
}

func (sp *SelectPlaylist) onSearched(query string) []*mediaprovider.SearchResult {
	var results []*mediaprovider.SearchResult
	if query != "" {
		var filteredPlaylists []*mediaprovider.Playlist
		if playlists, err := sp.mp.GetPlaylists(); err != nil {
			// TODO: surface this error to user
			log.Printf("error getting playlists: %s", err.Error())
			return results
		} else {
			filteredPlaylists = sharedutil.FilterSlice(playlists, func(playlist *mediaprovider.Playlist) bool {
				return strings.Contains(
					sanitize.Accents(strings.ToLower(playlist.Name)),
					sanitize.Accents(strings.ToLower(query)),
				) && playlist.Owner == sp.loggedInUser
			})
		}

		results = append(results, &mediaprovider.SearchResult{
			Name:  fmt.Sprintf("Create new playlist: %s", query),
			Type:  mediaprovider.ContentTypePlaylist,
			Query: query,
		})
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

	}
	return results
}

func (sp *SelectPlaylist) onUpdateSearchResult(sr *searchResult, result *mediaprovider.SearchResult) {
	if result.ID == "" {
		sr.secondary.Segments = []widget.RichTextSegment{}
		sr.secondary.Refresh()
		return
	}

	maybePluralize := func(s string, size int) string {
		if size != 1 {
			return s + "s"
		}
		return s
	}
	secondaryText := fmt.Sprintf("%d %s", result.Size, maybePluralize("track", result.Size))
	sr.secondary.Segments = []widget.RichTextSegment{
		&widget.TextSegment{
			Text:  result.Type.String(),
			Style: widget.RichTextStyle{SizeName: theme.SizeNameCaptionText, TextStyle: fyne.TextStyle{Bold: true}, Inline: true},
		},
	}
	if secondaryText != "" {
		sr.secondary.Segments = append(sr.secondary.Segments,
			&widget.TextSegment{
				Text:  " · ",
				Style: widget.RichTextStyle{SizeName: theme.SizeNameCaptionText, Inline: true},
			},
			&widget.TextSegment{
				Text:  secondaryText,
				Style: widget.RichTextStyle{SizeName: theme.SizeNameCaptionText, Inline: true},
			},
		)
	}
	sr.secondary.Refresh()
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
	return sp.SearchDialog.SearchEntry
}
