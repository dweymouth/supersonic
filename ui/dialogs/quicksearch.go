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
	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/ui/layouts"
	myTheme "github.com/dweymouth/supersonic/ui/theme"
	"github.com/dweymouth/supersonic/ui/widgets"
)

type QuickSearch struct {
	widget.BaseWidget

	OnDismiss    func()
	OnNavigateTo func(mediaprovider.ContentType, string)

	SearchEntry fyne.Focusable // exported so it can be focused by the Controller

	mp mediaprovider.MediaProvider
	im *backend.ImageManager

	resultsMutex  sync.RWMutex
	searchResults []*mediaprovider.SearchResult
	list          *widget.List
	selectedIndex int

	content *fyne.Container
}

func NewQuickSearch(mp mediaprovider.MediaProvider, im *backend.ImageManager) *QuickSearch {
	q := &QuickSearch{
		mp: mp,
		im: im,
	}
	q.ExtendBaseWidget(q)

	se := newQuickSearchEntry()
	se.OnSearched = q.onSearched
	se.OnSubmitted = func(_ string) {
		q.onSelected(q.selectedIndex)
	}
	se.OnTypedDown = func() {
		q.resultsMutex.RLock()
		if q.selectedIndex < len(q.searchResults)-1 {
			q.selectedIndex++
		}
		q.resultsMutex.RUnlock()
		q.list.Select(q.selectedIndex)
	}
	se.OnTypedUp = func() {
		q.resultsMutex.RLock()
		if q.selectedIndex > 0 {
			q.selectedIndex--
		}
		q.resultsMutex.RUnlock()
		q.list.Select(q.selectedIndex)
	}
	se.OnTypedEscape = q.onDismiss
	q.SearchEntry = se
	q.list = widget.NewList(
		func() int {
			q.resultsMutex.RLock()
			defer q.resultsMutex.RUnlock()
			return len(q.searchResults)
		},
		func() fyne.CanvasObject { return newQuickSearchResult(q) },
		func(lii widget.ListItemID, co fyne.CanvasObject) {
			var result *mediaprovider.SearchResult
			q.resultsMutex.RLock()
			if len(q.searchResults) > lii {
				result = q.searchResults[lii]
			}
			q.resultsMutex.RUnlock()
			qs := co.(*quickSearchResult)
			qs.index = lii
			qs.Update(result)
		},
	)

	dismissBtn := widget.NewButton("Close", q.onDismiss)
	title := widget.NewRichText(&widget.TextSegment{Text: "Quick Search", Style: boldStyle})
	title.Segments[0].(*widget.TextSegment).Style.Alignment = fyne.TextAlignCenter
	q.content = container.NewBorder(
		container.NewVBox(title, se),
		container.NewVBox(widget.NewSeparator(), container.NewHBox(layout.NewSpacer(), dismissBtn)),
		nil, nil, q.list)
	return q
}

func (q *QuickSearch) onDismiss() {
	if q.OnDismiss != nil {
		q.OnDismiss()
	}
}

func (q *QuickSearch) onSelected(idx int) {
	if q.OnNavigateTo == nil {
		return
	}
	q.resultsMutex.RLock()
	if len(q.searchResults) <= idx {
		q.resultsMutex.RUnlock()
		return
	}
	id := q.searchResults[idx].ID
	typ := q.searchResults[idx].Type
	q.resultsMutex.RUnlock()
	q.OnNavigateTo(typ, id)
}

func (q *QuickSearch) onSearched(query string) {
	var results []*mediaprovider.SearchResult
	if query != "" {
		if res, err := q.mp.SearchAll(query, 20); err != nil {
			log.Printf("Error searching: %s", err.Error())
		} else {
			results = res
		}
	}
	q.resultsMutex.Lock()
	q.searchResults = results
	q.resultsMutex.Unlock()
	q.list.Refresh()
	q.list.ScrollToTop()
	q.list.Select(0)
}

func (q *QuickSearch) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(q.content)
}

func (q *QuickSearch) MinSize() fyne.Size {
	return fyne.NewSize(400, 350)
}

type quickSearchResult struct {
	widget.BaseWidget

	parent *QuickSearch

	id          string
	index       int
	contentType mediaprovider.ContentType

	imageLoader backend.ThumbnailLoader

	image     *widgets.ImagePlaceholder
	title     *widget.Label
	secondary *widget.RichText

	content *fyne.Container
}

func newQuickSearchResult(parent *QuickSearch) *quickSearchResult {
	qs := &quickSearchResult{
		parent:    parent,
		image:     widgets.NewImagePlaceholder(myTheme.AlbumIcon, 50),
		title:     widget.NewLabel(""),
		secondary: widget.NewRichText(),
	}
	qs.title.Wrapping = fyne.TextTruncate
	qs.secondary.Wrapping = fyne.TextTruncate
	qs.ExtendBaseWidget(qs)
	qs.imageLoader = parent.im.NewThumbnailLoader(func(im image.Image) {
		qs.image.SetImage(im, false)
	})
	qs.imageLoader.OnBeforeLoad = func() {
		qs.image.SetImage(nil, false)
	}

	return qs
}

func (q *quickSearchResult) Update(result *mediaprovider.SearchResult) {
	if result == nil {
		return
	}
	if q.contentType == result.Type && q.id == result.ID {
		return // nothing to do
	}
	q.id = result.ID
	q.contentType = result.Type
	q.image.CenterIcon = placeholderIconForContentType(result.Type)
	q.imageLoader.Load(result.CoverID)
	q.title.SetText(result.Name)
	q.secondary.Segments = []widget.RichTextSegment{&widget.TextSegment{Text: result.Type.String()}}
	q.secondary.Refresh()
}

func (q *quickSearchResult) Tapped(_ *fyne.PointEvent) {
	q.parent.onSelected(q.index)
}

func (q *quickSearchResult) CreateRenderer() fyne.WidgetRenderer {
	if q.content == nil {
		q.content = container.NewBorder(nil, nil, container.NewCenter(q.image), nil,
			container.New(&layouts.VboxCustomPadding{ExtraPad: -15},
				q.title,
				q.secondary,
			))
	}
	return widget.NewSimpleRenderer(q.content)
}

func (q *quickSearchResult) Refresh() {
	q.BaseWidget.Refresh()
}

type quickSearchEntry struct {
	widgets.SearchEntry

	OnTypedUp     func()
	OnTypedDown   func()
	OnTypedEscape func()
}

func newQuickSearchEntry() *quickSearchEntry {
	q := &quickSearchEntry{}
	q.ExtendBaseWidget(q)
	q.SearchEntry.Init()
	return q
}

func (q *quickSearchEntry) TypedKey(e *fyne.KeyEvent) {
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
