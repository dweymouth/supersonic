package browsing

import (
	"log"
	"net/url"
	"strings"

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
	radios []*mediaprovider.RadioStation
	list   *RadioList

	titleDisp *widget.RichText
	container *fyne.Container
	searcher  *widgets.SearchEntry
}

func NewRadiosPage(contr *controller.Controller, rp mediaprovider.RadioProvider) *RadiosPage {
	return newRadiosPage(contr, rp, "", 0)
}

func newRadiosPage(contr *controller.Controller, rp mediaprovider.RadioProvider, searchText string, scrollPos float32) *RadiosPage {
	a := &RadiosPage{
		contr:     contr,
		rp:        rp,
		titleDisp: widget.NewRichTextWithText("Internet Radio Stations"),
	}
	a.ExtendBaseWidget(a)
	a.titleDisp.Segments[0].(*widget.TextSegment).Style.SizeName = theme.SizeNameHeadingText
	a.list = NewRadioList()
	a.searcher = widgets.NewSearchEntry()
	a.searcher.PlaceHolder = "Search page"
	a.searcher.OnSearched = a.onSearched
	a.searcher.Entry.Text = searchText
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

func (a *RadiosPage) Route() controller.Route {
	// TODO
	//return controller.RadiosRoute()
	return controller.Route{}
}

func (a *RadiosPage) Reload() {
	go a.load(false, 0)
}

func (a *RadiosPage) Save() SavedPage {
	return &savedrRadiosPage{
		contr:      a.contr,
		rp:         a.rp,
		searchText: a.searcher.Entry.Text,
		scrollPos:  a.list.list.GetScrollOffset(),
	}
}

type savedrRadiosPage struct {
	contr      *controller.Controller
	rp         mediaprovider.RadioProvider
	searchText string
	scrollPos  float32
}

func (s *savedrRadiosPage) Restore() Page {
	return newRadiosPage(s.contr, s.rp, s.searchText, s.scrollPos)
}

func (a *RadiosPage) buildContainer() {
	searchVbox := container.NewVBox(layout.NewSpacer(), a.searcher, layout.NewSpacer())
	a.container = container.New(&layout.CustomPaddedLayout{LeftPadding: 15, RightPadding: 15, TopPadding: 5, BottomPadding: 15},
		container.NewBorder(
			container.New(&layout.CustomPaddedLayout{LeftPadding: -5},
				container.NewHBox(a.titleDisp, layout.NewSpacer(), searchVbox)),
			nil, nil, nil, a.list))
}

func (a *RadiosPage) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}

type RadioList struct {
	widget.BaseWidget

	OnPlay func(string)

	radios []*mediaprovider.RadioStation

	columnsLayout *layouts.ColumnsLayout
	hdr           *widgets.ListHeader
	list          *widgets.FocusList
	container     *fyne.Container
}

type RadioListRow struct {
	widgets.FocusListRowBase

	Item *mediaprovider.RadioStation

	nameLabel    *widget.Label
	homePageLink *widget.Hyperlink
}

func NewRadioListRow(layout *layouts.ColumnsLayout) *RadioListRow {
	a := &RadioListRow{
		nameLabel:    widget.NewLabel(""),
		homePageLink: widget.NewHyperlink("", nil),
	}
	a.ExtendBaseWidget(a)
	a.nameLabel.Truncation = fyne.TextTruncateEllipsis
	a.homePageLink.Truncation = fyne.TextTruncateEllipsis
	a.Content = container.New(layout, a.nameLabel, a.homePageLink)
	return a
}

func NewRadioList() *RadioList {
	a := &RadioList{
		columnsLayout: layouts.NewColumnsLayout([]float32{-1, 125, 125}),
	}
	a.ExtendBaseWidget(a)
	a.hdr = widgets.NewListHeader([]widgets.ListColumn{
		{Text: "Name", Alignment: fyne.TextAlignLeading, CanToggleVisible: false},
		{Text: "Home Page", Alignment: fyne.TextAlignTrailing, CanToggleVisible: false}},
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
			if row.Item != a.radios[id] {
				row.EnsureUnfocused()
				row.ListItemID = id
				row.Item = a.radios[id]
				row.nameLabel.Text = row.Item.Name
				row.homePageLink.Text = row.Item.HomePageURL
				if u, err := url.Parse(row.Item.HomePageURL); err == nil {
					row.homePageLink.URL = u
				}
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
		a.OnPlay(item.ID)
	}
}

func (a *RadioList) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}
