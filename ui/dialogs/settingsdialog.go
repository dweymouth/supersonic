package dialogs

import (
	"math"
	"strconv"
	"supersonic/backend"
	"supersonic/player"
	"supersonic/ui/layouts"
	"supersonic/ui/util"
	"supersonic/ui/widgets"
	"unicode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

var boldStyle = widget.RichTextStyle{TextStyle: fyne.TextStyle{Bold: true}}

type SettingsDialog struct {
	widget.BaseWidget

	OnReplayGainSettingsChanged    func()
	OnAudioExclusiveSettingChanged func()
	OnAudioDeviceSettingChanged    func()
	OnDismiss                      func()

	config       *backend.Config
	audioDevices []player.AudioDevice

	content fyne.CanvasObject
}

// TODO: having this depend on the player package for the AudioDevice type is kinda gross. Refactor.
func NewSettingsDialog(config *backend.Config, audioDeviceList []player.AudioDevice) *SettingsDialog {
	s := &SettingsDialog{config: config, audioDevices: audioDeviceList}
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
	if !s.config.Scrobbling.Enabled {
		percentEntry.Disable()
	}

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
	if !s.config.Scrobbling.Enabled || s.config.Scrobbling.ThresholdTimeSeconds < 0 {
		durationEntry.Disable()
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
	if !s.config.Scrobbling.Enabled {
		durationEnabled.Disable()
	}

	scrobbleEnabled := widget.NewCheck("Send playback statistics to server", func(checked bool) {
		s.config.Scrobbling.Enabled = checked
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
	})
	scrobbleEnabled.Checked = s.config.Scrobbling.Enabled

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
	deviceList := make([]string, len(s.audioDevices))
	var selIndex int
	for i, dev := range s.audioDevices {
		deviceList[i] = dev.Description
		if dev.Name == s.config.LocalPlayback.AudioDeviceName {
			selIndex = i
		}
	}
	deviceSelect := widget.NewSelect(deviceList, nil)
	deviceSelect.SetSelectedIndex(selIndex)
	deviceSelect.OnChanged = func(_ string) {
		dev := s.audioDevices[deviceSelect.SelectedIndex()]
		s.config.LocalPlayback.AudioDeviceName = dev.Name
		if s.OnAudioDeviceSettingChanged != nil {
			s.OnAudioDeviceSettingChanged()
		}
	}

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

	preventClipping := widget.NewCheck("", func(checked bool) {
		s.config.ReplayGain.PreventClipping = checked
		s.onReplayGainSettingsChanged()
	})
	preventClipping.Checked = s.config.ReplayGain.PreventClipping

	audioExclusive := widget.NewCheck("Audio exclusive mode", func(checked bool) {
		s.config.LocalPlayback.AudioExclusive = checked
		s.onAudioExclusiveSettingsChanged()
	})
	audioExclusive.Checked = s.config.LocalPlayback.AudioExclusive

	return container.NewTabItem("Playback", container.NewVBox(
		container.New(&layouts.MaxPadLayout{PadTop: 5},
			container.New(layout.NewFormLayout(),
				widget.NewLabel("Audio device"), container.NewBorder(nil, nil, nil, util.NewHSpace(70), deviceSelect),
				layout.NewSpacer(), container.NewHBox(audioExclusive, layout.NewSpacer()),
			)),
		container.New(&layouts.MaxPadLayout{PadLeft: 15, PadRight: 15}, widget.NewSeparator()),
		widget.NewRichText(&widget.TextSegment{Text: "ReplayGain", Style: boldStyle}),
		container.New(layout.NewFormLayout(),
			widget.NewLabel("ReplayGain mode"), container.NewGridWithColumns(2, replayGainSelect),
			widget.NewLabel("ReplayGain preamp"), container.NewHBox(preampGain, widget.NewLabel("dB")),
			widget.NewLabel("Prevent clipping"), container.NewHBox(preventClipping, layout.NewSpacer()),
		),
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
