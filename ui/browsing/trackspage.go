package browsing

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/sharedutil"
	"github.com/dweymouth/supersonic/ui/controller"
	"github.com/dweymouth/supersonic/ui/theme"
	"github.com/dweymouth/supersonic/ui/util"
	"github.com/dweymouth/supersonic/ui/widgets"
	"slices"
)

type TracksPage struct {
	widget.BaseWidget

	tracksPageState

	nowPlayingID string

	title           *widget.RichText
	sortOrder       *sortOrderSelect
	searcher        *widgets.SearchEntry
	tracklist       *widgets.Tracklist
	loader          widgets.TracklistLoader
	searchTracklist *widgets.Tracklist
	searchLoader    widgets.TracklistLoader
	playRandom      *widget.Button
	container       *fyne.Container
}

type tracksPageState struct {
	searchText string
	widgetPool *util.WidgetPool
	contr      *controller.Controller
	conf       *backend.TracksPageConfig
	mp         mediaprovider.MediaProvider
	im         *backend.ImageManager
	sorterId   int
	canRate    bool
	canShare   bool
}

func NewTracksPage(contr *controller.Controller, conf *backend.TracksPageConfig, pool *util.WidgetPool, mp mediaprovider.MediaProvider, im *backend.ImageManager) *TracksPage {
	t := &TracksPage{tracksPageState: tracksPageState{contr: contr, conf: conf, widgetPool: pool, mp: mp, im: im}}
	t.ExtendBaseWidget(t)

	t.tracklist = t.obtainTracklist()
	_, t.canRate = mp.(mediaprovider.SupportsRating)
	_, t.canShare = mp.(mediaprovider.SupportsSharing)
	t.tracklist.Options = widgets.TracklistOptions{
		DisableSorting: true,
		DisableRating:  !t.canRate,
		DisableSharing: !t.canShare,
		AutoNumber:     true,
	}
	t.tracklist.SetVisibleColumns(conf.TracklistColumns)
	t.tracklist.OnVisibleColumnsChanged = func(cols []string) {
		t.conf.TracklistColumns = cols
		if t.searchTracklist != nil {
			t.searchTracklist.SetVisibleColumns(cols)
		}
	}
	contr.ConnectTracklistActions(t.tracklist)

	t.createTitleAndSort()

	t.searcher = widgets.NewSearchEntry()
	t.searcher.PlaceHolder = lang.L("Search page")
	t.searcher.OnSearched = t.OnSearched
	t.createContainer()
	t.Reload()
	return t
}

func (t *TracksPage) createTitleAndSort() {
	t.title = widget.NewRichTextWithText(lang.L("All Tracks"))
	sortOrders := t.mp.TrackSortOrders()
	if len(sortOrders) > 1 {
		sorts, selected := t.SortOrders()
		t.sortOrder = NewSortOrderSelect(sorts, t.onSortOrderChanged)
		t.sortOrder.SetSelectedIndex(selected)
		t.sorterId = 0
	}
	t.title.Segments[0].(*widget.TextSegment).Style.SizeName = widget.RichTextStyleHeading.SizeName
	t.playRandom = widget.NewButtonWithIcon(lang.L("Play random"), theme.ShuffleIcon, t.playRandomSongs)
}

func (t *TracksPage) SortOrders() ([]string, int) {
	orders := t.mp.TrackSortOrders()
	sortOrder := slices.Index(orders, t.conf.SortOrder)
	if sortOrder < 0 {
		sortOrder = 0
	}

	return util.LocalizeSlice(orders), sortOrder
}

func (t *TracksPage) onSortOrderChanged(_ string) {
	t.conf.SortOrder = t.mp.TrackSortOrders()[t.getSortOrderIdx()]
	t.Reload()
}

func (t *TracksPage) getSortOrderIdx() int {
	if t.sortOrder != nil {
		return t.sortOrder.SelectedIndex()
	}
	return 0
}

func (t *TracksPage) createContainer() {
	playRandomVbox := container.NewVBox(layout.NewSpacer(), t.playRandom, layout.NewSpacer())
	searchVbox := container.NewVBox(layout.NewSpacer(), t.searcher, layout.NewSpacer())
	var topRow *fyne.Container
	if t.sortOrder != nil {
		topRow = container.NewHBox(t.title, container.NewCenter(t.sortOrder), playRandomVbox, layout.NewSpacer(), searchVbox)
	} else {
		topRow = container.NewHBox(t.title, playRandomVbox, layout.NewSpacer(), searchVbox)
	}
	t.container = container.New(&layout.CustomPaddedLayout{LeftPadding: 15, RightPadding: 15, TopPadding: 5, BottomPadding: 15},
		container.NewBorder(topRow, nil, nil, nil, t.tracklist))
}

func (t *TracksPage) Route() controller.Route {
	return controller.TracksRoute()
}

func (t *TracksPage) Reload() {
	t.tracklist.Clear()
	iter := t.mp.IterateTracks(t.mp.TrackSortOrders()[t.getSortOrderIdx()], "")
	// loads asynchronously
	t.loader = widgets.NewTracklistLoader(t.tracklist, iter)
}

var _ CanShowNowPlaying = (*TracksPage)(nil)

func (t *TracksPage) OnSongChange(item mediaprovider.MediaItem, lastScrobbledIfAny *mediaprovider.Track) {
	t.nowPlayingID = sharedutil.MediaItemIDOrEmptyStr(item)
	t.tracklist.SetNowPlaying(t.nowPlayingID)
	if t.searchTracklist != nil {
		t.searchTracklist.SetNowPlaying(t.nowPlayingID)
	}
	playedID := sharedutil.MediaItemIDOrEmptyStr(lastScrobbledIfAny)
	t.tracklist.IncrementPlayCount(playedID)
	if t.searchTracklist != nil {
		t.searchTracklist.IncrementPlayCount(playedID)
	}
}

var _ Scrollable = (*TracksPage)(nil)

func (g *TracksPage) Scroll(scrollAmt float32) {
	g.tracklist.Scroll(scrollAmt)
}

var _ Searchable = (*TracksPage)(nil)

func (t *TracksPage) SearchWidget() fyne.Focusable {
	return t.searcher
}

func (t *TracksPage) OnSearched(query string) {
	t.searchText = query
	if query == "" {
		t.container.Objects[0].(*fyne.Container).Objects[0] = t.tracklist
		if t.searchTracklist != nil {
			t.searchTracklist.Clear()
		}
		t.Refresh()
		return
	}
	t.doSearch(query)
}

func (t *TracksPage) doSearch(query string) {
	if t.searchTracklist == nil {
		t.searchTracklist = t.obtainTracklist()
		t.searchTracklist.Options = widgets.TracklistOptions{
			AutoNumber:     true,
			DisableSorting: true,
			DisableRating:  !t.canRate,
			DisableSharing: !t.canShare,
		}
		t.searchTracklist.SetVisibleColumns(t.conf.TracklistColumns)
		t.searchTracklist.SetNowPlaying(t.nowPlayingID)
		t.searchTracklist.OnVisibleColumnsChanged = func(cols []string) {
			t.conf.TracklistColumns = cols
			t.tracklist.SetVisibleColumns(cols)
		}
		t.contr.ConnectTracklistActions(t.searchTracklist)
	} else {
		t.searchTracklist.Clear()
	}
	iter := t.mp.IterateTracks(t.mp.TrackSortOrders()[t.getSortOrderIdx()], query)
	t.searchLoader = widgets.NewTracklistLoader(t.searchTracklist, iter)
	t.container.Objects[0].(*fyne.Container).Objects[0] = t.searchTracklist
	t.Refresh()
}

func (t *TracksPage) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(t.container)
}

func (t *TracksPage) Save() SavedPage {
	t.loader.Dispose()
	t.tracklist.Clear()
	t.widgetPool.Release(util.WidgetTypeTracklist, t.tracklist)
	if t.searchTracklist != nil {
		t.searchLoader.Dispose()
		t.searchTracklist.Clear()
		t.widgetPool.Release(util.WidgetTypeTracklist, t.searchTracklist)
	}
	state := t.tracksPageState
	return &state
}

func (s *tracksPageState) Restore() Page {
	t := NewTracksPage(s.contr, s.conf, s.widgetPool, s.mp, s.im)
	t.searchText = s.searchText
	if t.searchText != "" {
		t.searcher.Entry.Text = t.searchText
		t.doSearch(t.searchText)
	}
	return t
}

func (t *TracksPage) playRandomSongs() {
	t.contr.App.PlaybackManager.PlayRandomSongs("")
}

func (t *TracksPage) obtainTracklist() *widgets.Tracklist {
	if tl := t.widgetPool.Obtain(util.WidgetTypeTracklist); tl != nil {
		tracklist := tl.(*widgets.Tracklist)
		tracklist.Reset()
		return tracklist
	}
	return widgets.NewTracklist(nil, t.im, false)
}
