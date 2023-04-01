package dialogs

import (
	"math"
	"strconv"
	"supersonic/backend"
	"supersonic/ui/widgets"
	"unicode"

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

	tabs := container.NewAppTabs(
		s.createGeneralTab(),
		s.createPlaybackTab(),
	)
	s.content = container.NewVBox(tabs, widget.NewSeparator(),
		container.NewHBox(layout.NewSpacer(), widget.NewButton("Close", func() {
			if s.OnDismiss != nil {
				s.OnDismiss()
			}
		})))

	return s
}

func (s *SettingsDialog) createGeneralTab() *container.TabItem {
	twoDigitValidator := func(text string, r rune) bool {
		return unicode.IsDigit(r) && len(text) < 2
	}

	percentEntry := widgets.NewTextRestrictedEntry(twoDigitValidator)
	percentEntry.SetMinCharWidth(2)
	percentEntry.OnChanged = func(str string) {
		if i, err := strconv.Atoi(str); err == nil {
			s.config.Scrobbling.ThresholdPercent = i
		}
	}
	percentEntry.Text = strconv.Itoa(s.config.Scrobbling.ThresholdPercent)

	durationEntry := widgets.NewTextRestrictedEntry(twoDigitValidator)
	durationEntry.SetMinCharWidth(2)
	durationEntry.OnChanged = func(str string) {
		if i, err := strconv.Atoi(str); err == nil {
			s.config.Scrobbling.ThresholdTimeSeconds = i * 60
		}
	}
	if secs := s.config.Scrobbling.ThresholdTimeSeconds; secs >= 0 {
		val := int(math.Round(float64(secs) / 60.))
		durationEntry.Text = strconv.Itoa(val)
	}

	lastScrobbleText := durationEntry.Text
	if lastScrobbleText == "" {
		lastScrobbleText = "4" // default scrobble minutes
	}
	durationEnabled := widget.NewCheck("or when", func(checked bool) {
		if !checked {
			s.config.Scrobbling.ThresholdTimeSeconds = -1
			lastScrobbleText = durationEntry.Text
			durationEntry.Text = ""
			durationEntry.Disable()
		} else {
			durationEntry.Text = lastScrobbleText
			durationEntry.Enable()
			durationEntry.Refresh()
			durationEntry.OnChanged(durationEntry.Text)
		}
	})
	durationEnabled.Checked = s.config.Scrobbling.ThresholdTimeSeconds >= 0

	scrobbleEnabled := widget.NewCheckWithData("Send playback statistics to server", binding.BindBool(&s.config.Scrobbling.Enabled))
	scrobbleEnabled.OnChanged = func(checked bool) {
		if !checked {
			percentEntry.Disable()
			durationEnabled.Disable()
			durationEntry.Disable()
		} else {
			percentEntry.Enable()
			durationEnabled.Enable()
			if durationEnabled.Checked {
				durationEntry.Enable()
			}
		}
	}

	return container.NewTabItem("General", container.NewVBox(
		widget.NewRichText(&widget.TextSegment{Text: "Scrobbling", Style: boldStyle}),
		scrobbleEnabled,
		container.NewHBox(
			widget.NewLabel("Scrobble when"),
			percentEntry,
			widget.NewLabel("percent of track is played"),
		),
		container.NewHBox(
			durationEnabled,
			durationEntry,
			widget.NewLabel("minutes of track have been played"),
		),
	))
}

func (s *SettingsDialog) createPlaybackTab() *container.TabItem {
	replayGainSelect := widget.NewSelect([]string{"None", "Album", "Track"}, nil)

	return container.NewTabItem("Playback", container.NewVBox(
		widget.NewRichText(&widget.TextSegment{Text: "ReplayGain", Style: boldStyle}),
		container.New(layout.NewFormLayout(),
			widget.NewLabel("ReplayGain mode"), replayGainSelect,
			widget.NewLabel("Prevent clipping"), widget.NewCheckWithData("", binding.BindBool(&s.config.ReplayGain.PreventClipping)),
		),
		widget.NewSeparator(),
		widget.NewCheckWithData("Audio exclusive mode", binding.BindBool(&s.config.LocalPlayback.AudioExclusive)),
	))
}

func (s *SettingsDialog) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(s.content)
}
