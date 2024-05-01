package dialogs

import (
	"image"
	"log"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"

	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/supersonic/backend/mediaprovider"

	"github.com/dweymouth/supersonic/ui/layouts"
	myTheme "github.com/dweymouth/supersonic/ui/theme"
	"github.com/dweymouth/supersonic/ui/util"
	"github.com/dweymouth/supersonic/ui/widgets"
)

type SearchDialog struct {
	widget.BaseWidget

	SearchEntry fyne.Focusable // exported so it can be focused by the Controller

	imgSource util.ImageFetcher

	resultsMutex  sync.RWMutex
	searchResults []*mediaprovider.SearchResult
	loadingDots   *widgets.LoadingDots
	list          *widget.List
	selectedIndex int

	content *fyne.Container

	OnDismiss             func()
	OnNavigateTo          func(mediaprovider.ContentType, string)
	OnSearched            func(string) []*mediaprovider.SearchResult
	OnUpdateSearchResults func(*searchResult, *mediaprovider.SearchResult)
}

func NewSearchDialog(im util.ImageFetcher, placeholderTitle string, onSearched func(string) []*mediaprovider.SearchResult, onUpdateSearchResult func(*searchResult, *mediaprovider.SearchResult)) *SearchDialog {
	sd := &SearchDialog{
		imgSource:             im,
		loadingDots:           widgets.NewLoadingDots(),
		OnSearched:            onSearched,
		OnUpdateSearchResults: onUpdateSearchResult,
	}
	sd.ExtendBaseWidget(sd)

	se := newSearchEntry()
	se.OnSearched = sd.onSearched
	se.OnSubmitted = func(_ string) {
		sd.onSelected(sd.selectedIndex)
	}
	se.OnTypedDown = sd.moveSelectionDown
	se.OnTypedUp = sd.moveSelectionUp
	se.OnTypedEscape = sd.onDismiss
	sd.SearchEntry = se
	sd.list = widget.NewList(
		func() int {
			sd.resultsMutex.RLock()
			defer sd.resultsMutex.RUnlock()
			return len(sd.searchResults)
		},
		func() fyne.CanvasObject { return newSearchResult(sd) },
		func(lii widget.ListItemID, co fyne.CanvasObject) {
			var result *mediaprovider.SearchResult
			sd.resultsMutex.RLock()
			if len(sd.searchResults) > lii {
				result = sd.searchResults[lii]
			}
			sd.resultsMutex.RUnlock()
			sr := co.(*searchResult)
			sr.index = lii
			sd.update(sr, result)
		},
	)

	dismissBtn := widget.NewButton("Close", sd.onDismiss)
	title := widget.NewRichText(&widget.TextSegment{Text: placeholderTitle, Style: util.BoldRichTextStyle})
	title.Segments[0].(*widget.TextSegment).Style.Alignment = fyne.TextAlignCenter
	sd.content = container.NewStack(
		container.NewBorder(
			container.NewVBox(title, se),
			container.NewVBox(widget.NewSeparator(), container.NewHBox(layout.NewSpacer(), dismissBtn)),
			nil, nil, sd.list),
		container.NewCenter(sd.loadingDots),
	)
	return sd
}

func (sd *SearchDialog) onDismiss() {
	if sd.OnDismiss != nil {
		sd.OnDismiss()
	}
}

func (sd *SearchDialog) onSelected(idx int) {
	if sd.OnNavigateTo == nil {
		return
	}
	sd.resultsMutex.RLock()
	if len(sd.searchResults) <= idx {
		sd.resultsMutex.RUnlock()
		return
	}
	id := sd.searchResults[idx].ID
	typ := sd.searchResults[idx].Type
	sd.resultsMutex.RUnlock()
	sd.OnNavigateTo(typ, id)
}

func (sd *SearchDialog) moveSelectionDown() {
	sd.resultsMutex.RLock()
	if sd.selectedIndex < len(sd.searchResults)-1 {
		sd.selectedIndex++
	}
	sd.resultsMutex.RUnlock()
	sd.list.Select(sd.selectedIndex)
}

func (sd *SearchDialog) moveSelectionUp() {
	sd.resultsMutex.RLock()
	if sd.selectedIndex > 0 {
		sd.selectedIndex--
	}
	sd.resultsMutex.RUnlock()
	sd.list.Select(sd.selectedIndex)
}

func (sd *SearchDialog) onSearched(query string) {
	sd.loadingDots.Start()
	var results []*mediaprovider.SearchResult
	if query != "" {
		res := sd.OnSearched(query)
		if len(res) == 0 {
			log.Println("No results matched the query.")
		} else {
			results = res
		}
	}
	sd.loadingDots.Stop()
	sd.resultsMutex.Lock()
	sd.searchResults = results
	sd.resultsMutex.Unlock()
	sd.list.Refresh()
	sd.list.ScrollToTop()
	sd.selectedIndex = 0
	sd.list.Select(0)
}

func (sd *SearchDialog) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(sd.content)
}

func (sd *SearchDialog) MinSize() fyne.Size {
	return fyne.NewSize(400, 350)
}

func (sd *SearchDialog) update(sr *searchResult, result *mediaprovider.SearchResult) {
	if result == nil {
		return
	}
	if sr.contentType == result.Type && sr.id == result.ID {
		return // nothing to do
	}
	sr.id = result.ID
	sr.contentType = result.Type
	sr.image.CenterIcon = placeholderIconForContentType(result.Type)
	sr.imageLoader.Load(result.CoverID)
	sr.title.SetText(result.Name)

	sd.OnUpdateSearchResults(sr, result)
}

func placeholderIconForContentType(c mediaprovider.ContentType) fyne.Resource {
	switch c {
	case mediaprovider.ContentTypeAlbum:
		return myTheme.AlbumIcon
	case mediaprovider.ContentTypeArtist:
		return myTheme.ArtistIcon
	case mediaprovider.ContentTypeTrack:
		return myTheme.TracksIcon
	case mediaprovider.ContentTypeGenre:
		return myTheme.GenreIcon
	case mediaprovider.ContentTypePlaylist:
		return myTheme.PlaylistIcon
	default:
		return theme.WarningIcon() // unreached
	}
}

type searchResult struct {
	widget.BaseWidget

	parent *SearchDialog

	id          string
	index       int
	contentType mediaprovider.ContentType

	imageLoader util.ThumbnailLoader

	image     *widgets.ImagePlaceholder
	title     *widget.Label
	secondary *widget.RichText

	content *fyne.Container
}

func newSearchResult(parent *SearchDialog) *searchResult {
	qs := &searchResult{
		parent:    parent,
		image:     widgets.NewImagePlaceholder(myTheme.AlbumIcon, 50),
		title:     widget.NewLabel(""),
		secondary: widget.NewRichText(),
	}
	qs.title.Truncation = fyne.TextTruncateEllipsis
	qs.secondary.Truncation = fyne.TextTruncateEllipsis
	qs.ExtendBaseWidget(qs)
	qs.imageLoader = util.NewThumbnailLoader(parent.imgSource, func(im image.Image) {
		qs.image.SetImage(im, false)
	})
	qs.imageLoader.OnBeforeLoad = func() {
		qs.image.SetImage(nil, false)
	}

	return qs
}

func (q *searchResult) Tapped(_ *fyne.PointEvent) {
	q.parent.onSelected(q.index)
}

func (q *searchResult) CreateRenderer() fyne.WidgetRenderer {
	if q.content == nil {
		q.content = container.NewBorder(nil, nil, container.NewCenter(q.image), nil,
			container.New(&layouts.VboxCustomPadding{ExtraPad: -15},
				q.title,
				q.secondary,
			))
	}
	return widget.NewSimpleRenderer(q.content)
}

func (q *searchResult) Refresh() {
	q.BaseWidget.Refresh()
}

type searchEntry struct {
	widgets.SearchEntry

	OnTypedUp     func()
	OnTypedDown   func()
	OnTypedEscape func()
}

func newSearchEntry() *searchEntry {
	q := &searchEntry{}
	q.ExtendBaseWidget(q)
	q.SearchEntry.Init()
	return q
}

func (q *searchEntry) TypedKey(e *fyne.KeyEvent) {
	switch {
	case e.Name == fyne.KeyUp && q.OnTypedUp != nil:
		q.OnTypedUp()
	case e.Name == fyne.KeyDown && q.OnTypedDown != nil:
		q.OnTypedDown()
	case e.Name == fyne.KeyEscape && q.OnTypedEscape != nil:
		q.OnTypedEscape()
	default:
		q.SearchEntry.TypedKey(e)
	}

}
