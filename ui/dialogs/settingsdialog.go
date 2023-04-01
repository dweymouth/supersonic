package dialogs

import (
	"math"
	"strconv"
	"supersonic/backend"
	"supersonic/ui/layouts"
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
	replayGainSelect.OnChanged = func(_ string) {
		switch replayGainSelect.SelectedIndex() {
		case 0:
			s.config.ReplayGain.Mode = backend.ReplayGainNone
		case 1:
			s.config.ReplayGain.Mode = backend.ReplayGainAlbum
		case 2:
			s.config.ReplayGain.Mode = backend.ReplayGainTrack
		}
		s.onReplayGainSettingsChanged()
	}

	// set initially selected option
	switch s.config.ReplayGain.Mode {
	case backend.ReplayGainAlbum:
		replayGainSelect.SetSelectedIndex(1)
	case backend.ReplayGainTrack:
		replayGainSelect.SetSelectedIndex(2)
	default:
		replayGainSelect.SetSelectedIndex(0)
	}

	preampGain := widgets.NewTextRestrictedEntry(func(curText string, r rune) bool {
		return (curText == "" && r == '-') ||
			(curText == "" && unicode.IsDigit(r)) ||
			((curText == "-" || curText == "0") && unicode.IsDigit(r))
	})
	preampGain.SetMinCharWidth(2)
	preampGain.OnChanged = func(text string) {
		if f, err := strconv.ParseFloat(text, 64); err == nil {
			s.config.ReplayGain.PreampGainDB = f
			s.onReplayGainSettingsChanged()
		}
	}
	initVal := math.Round(s.config.ReplayGain.PreampGainDB)
	if initVal < -9 {
		initVal = -9
	} else if initVal > 9 {
		initVal = 9
	}
	preampGain.Text = strconv.Itoa(int(initVal))

	preventClipping := widget.NewCheckWithData("", binding.BindBool(&s.config.ReplayGain.PreventClipping))
	preventClipping.OnChanged = func(_ bool) { s.onReplayGainSettingsChanged() }

	audioExclusive := widget.NewCheckWithData("Audio exclusive mode", binding.BindBool(&s.config.LocalPlayback.AudioExclusive))
	audioExclusive.OnChanged = func(_ bool) { s.onAudioExclusiveSettingsChanged() }

	return container.NewTabItem("Playback", container.NewVBox(
		widget.NewRichText(&widget.TextSegment{Text: "ReplayGain", Style: boldStyle}),
		container.New(layout.NewFormLayout(),
			widget.NewLabel("ReplayGain mode"), replayGainSelect,
			widget.NewLabel("ReplayGain preamp"), container.NewHBox(preampGain, widget.NewLabel("dB")),
			widget.NewLabel("Prevent clipping"), container.NewHBox(preventClipping, layout.NewSpacer()),
		),
		container.New(&layouts.MaxPadLayout{PadLeft: 15, PadRight: 15}, widget.NewSeparator()),
		container.NewHBox(audioExclusive, layout.NewSpacer()),
	))
}

func (s *SettingsDialog) onReplayGainSettingsChanged() {
	if s.OnReplayGainSettingsChanged != nil {
		s.OnReplayGainSettingsChanged()
	}
}

func (s *SettingsDialog) onAudioExclusiveSettingsChanged() {
	if s.OnAudioExclusiveSettingChanged != nil {
		s.OnAudioExclusiveSettingChanged()
	}
}

func (s *SettingsDialog) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(s.content)
}
