package dialogs

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/ui/util"
)

type QuickSearch struct {
	SearchDialog *SearchDialog
	mp           mediaprovider.MediaProvider
}

func NewQuickSearch(mp mediaprovider.MediaProvider, im util.ImageFetcher) *QuickSearch {
	q := &QuickSearch{mp: mp}
	q.SearchDialog = NewSearchDialog(im, lang.L("Search Everywhere"), lang.L("Close"), q.onSearched)
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

func (q *QuickSearch) SetOnDismiss(onDismiss func()) {
	q.SearchDialog.OnDismiss = onDismiss
}

func (q *QuickSearch) SetOnNavigateTo(onNavigateTo func(mediaprovider.ContentType, string)) {
	q.SearchDialog.OnNavigateTo = onNavigateTo
}

func (q *QuickSearch) MinSize() fyne.Size {
	return q.SearchDialog.MinSize()
}

func (q *QuickSearch) GetSearchEntry() fyne.Focusable {
	return q.SearchDialog.GetSearchEntry()
}
