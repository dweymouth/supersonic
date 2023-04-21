package browsing

import (
	"supersonic/backend"
	"supersonic/res"
	"supersonic/ui/controller"
	"supersonic/ui/layouts"
	"supersonic/ui/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type TracksPage struct {
	widget.BaseWidget

	tracksPageState

	title      *widget.RichText
	tracklist  *widgets.Tracklist
	loader     widgets.TracklistLoader
	playRandom *widget.Button
	container  *fyne.Container
}

type tracksPageState struct {
	contr *controller.Controller
	conf  *backend.TracksPageConfig
	lm    *backend.LibraryManager
}

func NewTracksPage(contr *controller.Controller, conf *backend.TracksPageConfig, lm *backend.LibraryManager) *TracksPage {
	t := &TracksPage{tracksPageState: tracksPageState{contr: contr, conf: conf, lm: lm}}
	t.ExtendBaseWidget(t)
	t.tracklist = widgets.NewTracklist(nil)
	t.tracklist.AutoNumber = true
	t.tracklist.SetVisibleColumns(conf.TracklistColumns)
	contr.ConnectTracklistActions(t.tracklist)
	t.title = widget.NewRichTextWithText("All Tracks")
	t.title.Segments[0].(*widget.TextSegment).Style.SizeName = widget.RichTextStyleHeading.SizeName
	t.playRandom = widget.NewButtonWithIcon("Play random", res.ResShuffleInvertSvg, t.playRandomSongs)
	t.createContainer()
	t.Reload()
	return t
}

func (t *TracksPage) createContainer() {
	playRandomVbox := container.NewVBox(layout.NewSpacer(), t.playRandom, layout.NewSpacer())
	topRow := container.NewHBox(t.title, playRandomVbox, layout.NewSpacer()) //searchVbox, util.NewHSpace(15))
	t.container = container.New(&layouts.MaxPadLayout{PadLeft: 15, PadRight: 15, PadTop: 5, PadBottom: 15},
		container.NewBorder(topRow, nil, nil, nil, t.tracklist))
}

func (t *TracksPage) Route() controller.Route {
	return controller.TracksRoute()
}

func (t *TracksPage) Reload() {
	t.tracklist.Clear()
	iter := t.lm.AllTracksIterator()
	// loads asynchronously
	t.loader = widgets.NewTracklistLoader(t.tracklist, iter)
}

func (t *TracksPage) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(t.container)
}

func (t *TracksPage) Save() SavedPage {
	state := t.tracksPageState
	return &state
}

func (s *tracksPageState) Restore() Page {
	return NewTracksPage(s.contr, s.conf, s.lm)
}

func (t *TracksPage) playRandomSongs() {
	t.contr.App.PlaybackManager.PlayRandomSongs("")
}
