package dialogs

import (
	"fmt"
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

type QuickSearch struct {
	widget.BaseWidget

	OnDismiss    func()
	OnNavigateTo func(mediaprovider.ContentType, string)

	SearchEntry fyne.Focusable // exported so it can be focused by the Controller

	mp        mediaprovider.MediaProvider
	imgSource util.ImageFetcher

	resultsMutex  sync.RWMutex
	searchResults []*mediaprovider.SearchResult
	loadingDots   *widgets.LoadingDots
	list          *widget.List
	selectedIndex int

	content *fyne.Container
}

func NewQuickSearch(mp mediaprovider.MediaProvider, im util.ImageFetcher) *QuickSearch {
	q := &QuickSearch{
		mp:          mp,
		imgSource:   im,
		loadingDots: widgets.NewLoadingDots(),
	}
	q.ExtendBaseWidget(q)

	se := newQuickSearchEntry()
	se.OnSearched = q.onSearched
	se.OnSubmitted = func(_ string) {
		q.onSelected(q.selectedIndex)
	}
	se.OnTypedDown = q.moveSelectionDown
	se.OnTypedUp = q.moveSelectionUp
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
	title := widget.NewRichText(&widget.TextSegment{Text: "Quick Search", Style: util.BoldRichTextStyle})
	title.Segments[0].(*widget.TextSegment).Style.Alignment = fyne.TextAlignCenter
	q.content = container.NewStack(
		container.NewBorder(
			container.NewVBox(title, se),
			container.NewVBox(widget.NewSeparator(), container.NewHBox(layout.NewSpacer(), dismissBtn)),
			nil, nil, q.list),
		container.NewCenter(q.loadingDots),
	)
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

func (q *QuickSearch) moveSelectionDown() {
	q.resultsMutex.RLock()
	if q.selectedIndex < len(q.searchResults)-1 {
		q.selectedIndex++
	}
	q.resultsMutex.RUnlock()
	q.list.Select(q.selectedIndex)
}

func (q *QuickSearch) moveSelectionUp() {
	q.resultsMutex.RLock()
	if q.selectedIndex > 0 {
		q.selectedIndex--
	}
	q.resultsMutex.RUnlock()
	q.list.Select(q.selectedIndex)
}

func (q *QuickSearch) onSearched(query string) {
	q.loadingDots.Start()
	var results []*mediaprovider.SearchResult
	if query != "" {
		if res, err := q.mp.SearchAll(query, 20); err != nil {
			log.Printf("Error searching: %s", err.Error())
		} else {
			results = res
		}
	}
	q.loadingDots.Stop()
	q.resultsMutex.Lock()
	q.searchResults = results
	q.resultsMutex.Unlock()
	q.list.Refresh()
	q.list.ScrollToTop()
	q.selectedIndex = 0
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

	imageLoader util.ThumbnailLoader

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
	q.secondary.Segments = []widget.RichTextSegment{
		&widget.TextSegment{
			Text:  result.Type.String(),
			Style: widget.RichTextStyle{SizeName: theme.SizeNameCaptionText, TextStyle: fyne.TextStyle{Bold: true}, Inline: true},
		},
	}
	if secondaryText != "" {
		q.secondary.Segments = append(q.secondary.Segments,
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
