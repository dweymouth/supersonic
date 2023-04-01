package dialogs

import (
	"supersonic/backend"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

var boldStyle = widget.RichTextStyle{TextStyle: fyne.TextStyle{Bold: true}}

type SettingsDialog struct {
	widget.BaseWidget

	OnReplayGainSettingsChanged    func()
	OnAudioExclusiveSettingChanged func()
	OnDismiss                      func()

	config *backend.Config

	content fyne.CanvasObject
}

func NewSettingsDialog(config *backend.Config) *SettingsDialog {
	s := &SettingsDialog{config: config}
	s.ExtendBaseWidget(s)

	replayGainSelect := widget.NewSelect([]string{"None", "Album", "Track"}, nil)

	tabs := container.NewAppTabs(
		container.NewTabItem("General", container.NewVBox(
			widget.NewRichText(&widget.TextSegment{Text: "Scrobbling", Style: boldStyle}),
			widget.NewCheckWithData("Send playback statistics to server", binding.BindBool(&config.Scrobbling.Enabled)))),
		container.NewTabItem("Playback", container.NewVBox(
			widget.NewRichText(&widget.TextSegment{Text: "ReplayGain", Style: boldStyle}),
			container.New(layout.NewFormLayout(),
				widget.NewLabel("ReplayGain mode"), replayGainSelect,
				widget.NewLabel("Prevent clipping"), widget.NewCheckWithData("", binding.BindBool(&config.ReplayGain.PreventClipping)),
			),
			widget.NewSeparator(),
			widget.NewCheckWithData("Audio exclusive mode", binding.BindBool(&config.LocalPlayback.AudioExclusive)))),
	)
	s.content = container.NewVBox(tabs, widget.NewSeparator(),
		container.NewHBox(layout.NewSpacer(), widget.NewButton("Close", func() {
			if s.OnDismiss != nil {
				s.OnDismiss()
			}
		})))

	return s
}

func (s *SettingsDialog) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(s.content)
}
