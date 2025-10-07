package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/ui/controller"
	"github.com/dweymouth/supersonic/ui/theme"
	"github.com/dweymouth/supersonic/ui/widgets"
)

type Sidebar struct {
	widget.BaseWidget

	queueList *widgets.PlayQueueList
}

func NewSidebar(contr *controller.Controller, im *backend.ImageManager) *Sidebar {
	s := &Sidebar{
		queueList: widgets.NewPlayQueueList(im, false),
	}
	s.queueList.Reorderable = true
	contr.ConnectPlayQueuelistActions(s.queueList)
	s.ExtendBaseWidget(s)
	return s
}

func (s *Sidebar) SetQueueTracks(items []mediaprovider.MediaItem) {
	s.queueList.SetItems(items)
}

func (s *Sidebar) SetNowPlaying(itemID string) {
	s.queueList.SetNowPlaying(itemID)
}

func (s *Sidebar) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewStack(
		theme.NewThemedRectangle(theme.ColorNamePageBackground),
		container.NewAppTabs(
			container.NewTabItem(lang.L("Play Queue"), container.NewPadded(s.queueList)),
		),
	))
}
