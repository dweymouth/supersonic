package dialogs

import (
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/ui/util"
)

type QuickSearch struct {
	SearchDialog *SearchDialog
	mp           mediaprovider.MediaProvider
}

func NewQuickSearch(mp mediaprovider.MediaProvider, im util.ImageFetcher) *QuickSearch {

	q := &QuickSearch{
		mp: mp,
	}

	sd := NewSearchDialog(
		im,
		"Quick Search",
		q.onSearched,
		q.onUpdateSearchResult,
	)
	q.SearchDialog = sd
	return q
}

func (q *QuickSearch) onSearched(query string) []*mediaprovider.SearchResult {
	var results []*mediaprovider.SearchResult
	if query != "" {
		if res, err := q.mp.SearchAll(query, 20); err != nil {
			log.Printf("Error searching: %s", err.Error())
		} else {
			results = res
		}
	}
	return results
}

func (q *QuickSearch) onUpdateSearchResult(sr *searchResult, result *mediaprovider.SearchResult) {

	maybePluralize := func(s string, size int) string {
		if size != 1 {
			return s + "s"
		}
		return s
	}

	var secondaryText string
	switch result.Type {
	case mediaprovider.ContentTypeAlbum:
		secondaryText = result.ArtistName
	case mediaprovider.ContentTypeArtist:
		secondaryText = fmt.Sprintf("%d %s", result.Size, maybePluralize("album", result.Size))
	case mediaprovider.ContentTypeTrack:
		secondaryText = result.ArtistName
	case mediaprovider.ContentTypePlaylist:
		secondaryText = fmt.Sprintf("%d %s", result.Size, maybePluralize("track", result.Size))
	case mediaprovider.ContentTypeGenre:
		if result.Size > 0 {
			secondaryText = fmt.Sprintf("%d %s", result.Size, maybePluralize("album", result.Size))
		} else {
			secondaryText = ""
		}
	}
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

func (q *QuickSearch) SetOnDismiss(onDismiss func()) {
	q.SearchDialog.OnDismiss = onDismiss
}

func (q *QuickSearch) SetOnNavigateTo(onNavigateTo func(mediaprovider.ContentType, string)) {
	q.SearchDialog.OnNavigateTo = onNavigateTo
}

func (q *QuickSearch) GetMinSize() fyne.Size {
	return q.SearchDialog.MinSize()
}

func (q *QuickSearch) GetSearchEntry() fyne.Focusable {
	return q.SearchDialog.SearchEntry
}
