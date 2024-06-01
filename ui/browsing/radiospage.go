package browsing

import (
	"log"
	"net/url"
	"strings"

	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/sharedutil"
	"github.com/dweymouth/supersonic/ui/controller"
	"github.com/dweymouth/supersonic/ui/layouts"
	"github.com/dweymouth/supersonic/ui/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var _ fyne.Widget = (*ArtistPage)(nil)

type RadiosPage struct {
	widget.BaseWidget

	contr  *controller.Controller
	rp     mediaprovider.RadioProvider
	pm     *backend.PlaybackManager
	radios []*mediaprovider.RadioStation
	list   *RadioList

	nowPlayingID string

	titleDisp   *widget.RichText
	noRadiosMsg fyne.CanvasObject
	container   *fyne.Container
	searcher    *widgets.SearchEntry
}

func NewRadiosPage(contr *controller.Controller, rp mediaprovider.RadioProvider, pm *backend.PlaybackManager) *RadiosPage {
	return newRadiosPage(contr, rp, pm, "", 0)
}

func newRadiosPage(contr *controller.Controller, rp mediaprovider.RadioProvider, pm *backend.PlaybackManager, searchText string, scrollPos float32) *RadiosPage {
	a := &RadiosPage{
		contr:     contr,
		rp:        rp,
		pm:        pm,
		titleDisp: widget.NewRichTextWithText("Internet Radio Stations"),
	}
	a.ExtendBaseWidget(a)
	a.titleDisp.Segments[0].(*widget.TextSegment).Style.SizeName = theme.SizeNameHeadingText
	a.list = NewRadioList(&a.nowPlayingID)
	a.list.OnPlay = func(station *mediaprovider.RadioStation) {
		pm.LoadRadioStation(station, backend.Replace)
		pm.PlayFromBeginning()
	}
	a.searcher = widgets.NewSearchEntry()
	a.searcher.PlaceHolder = "Search page"
	a.searcher.OnSearched = a.onSearched
	a.searcher.Entry.Text = searchText

	a.noRadiosMsg = container.NewCenter(
		container.New(layout.NewCustomPaddedVBoxLayout(-10),
			container.NewCenter(
				container.NewBorder(nil, nil,
					widget.NewIcon(theme.InfoIcon()), nil,
					widget.NewRichText(&widget.TextSegment{
						Text:  "No radio stations available",
						Style: widget.RichTextStyleSubHeading,
					}))),
			widget.NewLabel("Configure your music server to add radio stations"),
		),
	)
	a.noRadiosMsg.Hide()

	a.buildContainer()
	go a.load(searchText != "", scrollPos)
	return a
}

// should be called asynchronously
func (a *RadiosPage) load(searchOnLoad bool, scrollPos float32) {
	radios, err := a.rp.GetRadioStations()
	if err != nil {
		log.Printf("error loading radios: %v", err.Error())
	}
	if len(radios) == 0 {
		a.noRadiosMsg.Show()
	} else {
		a.noRadiosMsg.Hide()
	}
	a.radios = radios
	if searchOnLoad {
		a.onSearched(a.searcher.Entry.Text)
		if scrollPos != 0 {
			a.list.list.ScrollToOffset(scrollPos)
		}
	} else {
		a.list.SetRadios(a.radios)
		if scrollPos != 0 {
			a.list.list.ScrollToOffset(scrollPos)
			return
		}
		a.list.Refresh()
	}
}

func (a *RadiosPage) onSearched(query string) {
	// since the radios list is returned in full non-paginated, we will do our own
	// simple search based on the radio name, rather than calling a server API
	if query == "" {
		a.list.SetRadios(a.radios)
	} else {
		query = strings.ToLower(query)
		result := sharedutil.FilterSlice(a.radios, func(x *mediaprovider.RadioStation) bool {
			return strings.Contains(strings.ToLower(x.Name), query)
		})
		a.list.SetRadios(result)
	}
	a.list.list.ScrollTo(0)
}

var _ Searchable = (*RadiosPage)(nil)

func (a *RadiosPage) SearchWidget() fyne.Focusable {
	return a.searcher
}

var _ Scrollable = (*RadiosPage)(nil)

func (a *RadiosPage) Scroll(amount float32) {
	a.list.list.ScrollToOffset(a.list.list.GetScrollOffset() + amount)
}

var _ CanShowNowPlaying = (*RadiosPage)(nil)

func (a *RadiosPage) OnSongChange(playing mediaprovider.MediaItem, _ *mediaprovider.Track) {
	if playing != nil {
		a.nowPlayingID = playing.Metadata().ID
	} else {
		a.nowPlayingID = ""
	}
	a.list.Refresh()
}

func (a *RadiosPage) Route() controller.Route {
	return controller.RadiosRoute()
}

func (a *RadiosPage) Reload() {
	go a.load(false, 0)
}

func (a *RadiosPage) Save() SavedPage {
	return &savedrRadiosPage{
		contr:      a.contr,
		rp:         a.rp,
		pm:         a.pm,
		searchText: a.searcher.Entry.Text,
		scrollPos:  a.list.list.GetScrollOffset(),
	}
}

type savedrRadiosPage struct {
	contr      *controller.Controller
	rp         mediaprovider.RadioProvider
	pm         *backend.PlaybackManager
	searchText string
	scrollPos  float32
}

func (s *savedrRadiosPage) Restore() Page {
	return newRadiosPage(s.contr, s.rp, s.pm, s.searchText, s.scrollPos)
}

func (a *RadiosPage) buildContainer() {
	searchVbox := container.NewVBox(layout.NewSpacer(), a.searcher, layout.NewSpacer())
	a.container = container.New(&layout.CustomPaddedLayout{LeftPadding: 15, RightPadding: 15, TopPadding: 5, BottomPadding: 15},
		container.NewBorder(
			container.New(&layout.CustomPaddedLayout{LeftPadding: -5},
				container.NewHBox(a.titleDisp, layout.NewSpacer(), searchVbox)),
			nil, nil, nil,
			container.NewStack(a.noRadiosMsg, a.list)),
	)
}

func (a *RadiosPage) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}

type RadioList struct {
	widget.BaseWidget

	OnPlay func(*mediaprovider.RadioStation)

	radios []*mediaprovider.RadioStation

	columnsLayout *layouts.ColumnsLayout
	hdr           *widgets.ListHeader
	list          *widgets.FocusList
	container     *fyne.Container
	playingIcon   fyne.CanvasObject
}

type RadioListRow struct {
	widgets.FocusListRowBase

	Item      *mediaprovider.RadioStation
	IsPlaying bool

	nameLabel    *widget.RichText
	homePageLink *widget.Hyperlink
}

func NewRadioListRow(layout *layouts.ColumnsLayout) *RadioListRow {
	a := &RadioListRow{
		nameLabel:    widget.NewRichTextWithText(""),
		homePageLink: widget.NewHyperlink("", nil),
	}
	a.ExtendBaseWidget(a)
	a.nameLabel.Truncation = fyne.TextTruncateEllipsis
	a.homePageLink.Truncation = fyne.TextTruncateEllipsis
	a.Content = container.New(layout, a.nameLabel, a.homePageLink)
	return a
}

func NewRadioList(nowPlayingIDPtr *string) *RadioList {
	a := &RadioList{
		columnsLayout: layouts.NewColumnsLayout([]float32{-1, -1}),
	}
	playIcon := theme.NewThemedResource(theme.MediaPlayIcon())
	playIcon.ColorName = theme.ColorNamePrimary
	a.playingIcon = container.NewCenter(widget.NewIcon(playIcon))
	a.ExtendBaseWidget(a)
	a.hdr = widgets.NewListHeader([]widgets.ListColumn{
		{Text: "Name", Alignment: fyne.TextAlignLeading, CanToggleVisible: false},
		{Text: "Home Page", Alignment: fyne.TextAlignLeading, CanToggleVisible: false}},
		a.columnsLayout)
	a.hdr.DisableSorting = true
	a.list = widgets.NewFocusList(
		func() int { return len(a.radios) },
		func() fyne.CanvasObject {
			r := NewRadioListRow(a.columnsLayout)
			r.OnDoubleTapped = func() { a.onPlayRadio(r.Item) }
			r.OnFocusNeighbor = func(up bool) {
				a.list.FocusNeighbor(r.ItemID(), up)
			}
			return r
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			row := item.(*RadioListRow)
			a.list.SetItemForID(id, row)
			changed := false
			if row.Item != a.radios[id] {
				row.EnsureUnfocused()
				row.ListItemID = id
				row.Item = a.radios[id]
				row.nameLabel.Segments[0].(*widget.TextSegment).Text = row.Item.Name
				row.homePageLink.Text = row.Item.HomePageURL
				if u, err := url.Parse(row.Item.HomePageURL); err == nil {
					row.homePageLink.URL = u
				}
				changed = true
			}
			isPlaying := *nowPlayingIDPtr == row.Item.ID
			if row.IsPlaying != isPlaying {
				row.IsPlaying = isPlaying
				row.nameLabel.Segments[0].(*widget.TextSegment).Style.TextStyle.Bold = isPlaying
				if isPlaying {
					row.Content.(*fyne.Container).Objects[0] =
						container.NewBorder(nil, nil, a.playingIcon, nil,
							container.New(layout.NewCustomPaddedLayout(0, 0, -5, 0), row.nameLabel))
				} else {
					row.Content.(*fyne.Container).Objects[0] = row.nameLabel
				}
				changed = true
			}
			if changed {
				row.Refresh()
			}
		},
	)
	a.container = container.NewBorder(a.hdr, nil, nil, nil, a.list)
	return a
}

func (g *RadioList) SetRadios(radios []*mediaprovider.RadioStation) {
	g.radios = radios
	g.Refresh()
}

func (a *RadioList) onPlayRadio(item *mediaprovider.RadioStation) {
	if a.OnPlay != nil {
		a.OnPlay(item)
	}
}

func (a *RadioList) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}
