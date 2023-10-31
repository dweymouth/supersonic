package dialogs

import (
	"image"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	myTheme "github.com/dweymouth/supersonic/ui/theme"
	"github.com/dweymouth/supersonic/ui/widgets"
)

type QuickSearch struct {
	widget.BaseWidget

	mp *mediaprovider.MediaProvider
	im *backend.ImageManager

	resultsMutex  sync.RWMutex
	searchResults []*mediaprovider.SearchResult
	list          *widget.List
	selectedIndex int

	content *fyne.Container
}

func NewQuickSearch(mp *mediaprovider.MediaProvider, im *backend.ImageManager) *QuickSearch {
	q := &QuickSearch{
		mp: mp,
		im: im,
	}
	q.ExtendBaseWidget(q)

	searchEntry := widgets.NewSearchEntry()
	searchEntry.OnChanged = q.onSearched
	q.list = widget.NewList(
		func() int {
			q.resultsMutex.RLock()
			defer q.resultsMutex.RUnlock()
			return len(q.searchResults)
		},
		func() fyne.CanvasObject { return newQuickSearchResult(im) },
		func(lii widget.ListItemID, co fyne.CanvasObject) {
			var result *mediaprovider.SearchResult
			q.resultsMutex.RLock()
			if len(q.searchResults) > lii {
				result = q.searchResults[lii]
			}
			q.resultsMutex.RUnlock()
			co.(*quickSearchResult).Update(result)
		},
	)
	title := widget.NewRichText(&widget.TextSegment{Text: "Quick Search", Style: boldStyle})
	title.Segments[0].(*widget.TextSegment).Style.Alignment = fyne.TextAlignCenter
	q.content = container.NewVBox(
		title,
		container.NewBorder(searchEntry, nil, nil, nil, q.list),
	)
	return q
}

func (q *QuickSearch) onSearched(query string) {
}

func (q *QuickSearch) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(q.content)
}

type quickSearchResult struct {
	widget.BaseWidget

	//	parent *QuickSearch

	imageLoader backend.ThumbnailLoader

	image     *widgets.ImagePlaceholder
	title     *widget.Label
	secondary *widget.RichText
	selection *canvas.Rectangle

	content *fyne.Container
}

func newQuickSearchResult(im *backend.ImageManager) *quickSearchResult {
	qs := &quickSearchResult{
		image:     widgets.NewImagePlaceholder(myTheme.AlbumIcon, 64),
		title:     widget.NewLabel(""),
		secondary: widget.NewRichText(),
	}
	qs.ExtendBaseWidget(qs)
	qs.imageLoader = im.NewThumbnailLoader(func(im image.Image) {
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
	q.image.CenterIcon = placeholderIconForContentType(result.Type)
	q.imageLoader.Load(result.CoverID)
	q.title.SetText(result.Name)
	//TODO: q.secondary.Segments =
}

func (q *quickSearchResult) CreateRenderer() fyne.WidgetRenderer {
	if q.selection == nil {
		q.selection = canvas.NewRectangle(theme.SelectionColor())
	}
	if q.content == nil {
		q.content = container.NewMax(
			q.selection,
			container.NewBorder(nil, nil, q.image, nil,
				container.NewVBox(
					q.title,
					q.secondary,
				)),
		)
	}
	return widget.NewSimpleRenderer(q.content)
}

func (q *quickSearchResult) Refresh() {
	q.BaseWidget.Refresh()
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
