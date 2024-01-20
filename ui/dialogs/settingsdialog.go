package dialogs

import (
	"errors"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/backend/player/mpv"
	"github.com/dweymouth/supersonic/ui/layouts"
	myTheme "github.com/dweymouth/supersonic/ui/theme"
	"github.com/dweymouth/supersonic/ui/util"
	"github.com/dweymouth/supersonic/ui/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type SettingsDialog struct {
	widget.BaseWidget

	OnReplayGainSettingsChanged    func()
	OnAudioExclusiveSettingChanged func()
	OnAudioDeviceSettingChanged    func()
	OnThemeSettingChanged          func()
	OnDismiss                      func()
	OnEqualizerSettingsChanged     func()

	config       *backend.Config
	audioDevices []mpv.AudioDevice
	themeFiles   map[string]string // filename -> displayName
	promptText   *widget.RichText

	clientDecidesScrobble bool

	content fyne.CanvasObject
}

// TODO: having this depend on the mpv package for the AudioDevice type is kinda gross. Refactor.
func NewSettingsDialog(
	config *backend.Config,
	audioDeviceList []mpv.AudioDevice,
	themeFileList map[string]string,
	equalizerBands []string,
	clientDecidesScrobble bool,
	isLocalPlayer bool,
	isReplayGainPlayer bool,
	isEqualizerPlayer bool,
	window fyne.Window,
) *SettingsDialog {
	s := &SettingsDialog{config: config, audioDevices: audioDeviceList, themeFiles: themeFileList, clientDecidesScrobble: clientDecidesScrobble}
	s.ExtendBaseWidget(s)

	// TODO: Once Fyne supports disableable sliders, it's probably a nicer UX
	// to create the equalizer tab but disable it if we are not using an equalizer player
	var tabs *container.AppTabs
	if isEqualizerPlayer {
		tabs = container.NewAppTabs(
			s.createGeneralTab(),
			s.createPlaybackTab(isLocalPlayer, isReplayGainPlayer),
			s.createEqualizerTab(equalizerBands),
			s.createExperimentalTab(window),
		)
	} else {
		tabs = container.NewAppTabs(
			s.createGeneralTab(),
			s.createPlaybackTab(isLocalPlayer, isReplayGainPlayer),
			s.createExperimentalTab(window),
		)
	}

	tabs.SelectIndex(s.getActiveTabNumFromConfig())
	// workaround issue where inactivated tabs don't fully update when theme setting is changed
	tabs.OnSelected = func(ti *container.TabItem) {
		ti.Content.Refresh()
		s.saveSelectedTab(tabs.SelectedIndex())
	}
	s.promptText = widget.NewRichTextWithText("")
	s.content = container.NewVBox(tabs, widget.NewSeparator(),
		container.NewHBox(s.promptText, layout.NewSpacer(), widget.NewButton("Close", func() {
			if s.OnDismiss != nil {
				s.OnDismiss()
			}
		})))

	return s
}

func (s *SettingsDialog) createGeneralTab() *container.TabItem {
	themeNames := []string{"Default"}
	themeFileNames := []string{""}
	i, selIndex := 1, 0
	for filename, displayname := range s.themeFiles {
		themeFileNames = append(themeFileNames, filename)
		themeNames = append(themeNames, displayname)
		if strings.EqualFold(filename, s.config.Theme.ThemeFile) {
			selIndex = i
		}
		i++
	}

	themeFileSelect := widget.NewSelect(themeNames, nil)
	themeFileSelect.SetSelectedIndex(selIndex)
	themeFileSelect.OnChanged = func(_ string) {
		s.config.Theme.ThemeFile = themeFileNames[themeFileSelect.SelectedIndex()]
		if s.OnThemeSettingChanged != nil {
			s.OnThemeSettingChanged()
		}
	}
	themeModeSelect := widget.NewSelect([]string{
		string(myTheme.AppearanceDark),
		string(myTheme.AppearanceLight),
		string(myTheme.AppearanceAuto)}, nil)
	themeModeSelect.OnChanged = func(_ string) {
		s.config.Theme.Appearance = themeModeSelect.Options[themeModeSelect.SelectedIndex()]
		if s.OnThemeSettingChanged != nil {
			s.OnThemeSettingChanged()
		}
	}
	themeModeSelect.SetSelected(s.config.Theme.Appearance)
	if themeModeSelect.Selected == "" {
		themeModeSelect.SetSelectedIndex(0)
	}

	startupPage := widget.NewSelect(backend.SupportedStartupPages, func(choice string) {
		s.config.Application.StartupPage = choice
	})
	startupPage.SetSelected(s.config.Application.StartupPage)
	if startupPage.Selected == "" {
		startupPage.SetSelectedIndex(0)
	}
	closeToTray := widget.NewCheckWithData("Close to system tray",
		binding.BindBool(&s.config.Application.CloseToSystemTray))
	if !s.config.Application.EnableSystemTray {
		closeToTray.Disable()
	}
	systemTrayEnable := widget.NewCheck("Enable system tray", func(val bool) {
		s.config.Application.EnableSystemTray = val
		// TODO: see https://github.com/fyne-io/fyne/issues/3788
		// Once Fyne supports removing/hiding an existing system tray menu,
		// the restart required prompt can be removed and this dialog
		// can expose a callback for the Controller to show/hide the system tray menu.
		s.setRestartRequired()
		if val {
			closeToTray.Enable()
		} else {
			closeToTray.Disable()
		}
	})
	systemTrayEnable.Checked = s.config.Application.EnableSystemTray

	saveQueue := widget.NewCheckWithData("Save play queue on exit",
		binding.BindBool(&s.config.Application.SavePlayQueue))
	trackNotif := widget.NewCheckWithData("Show notification on track change",
		binding.BindBool(&s.config.Application.ShowTrackChangeNotification))

	// Scrobble settings

	twoDigitValidator := func(text, selText string, r rune) bool {
		return unicode.IsDigit(r) && len(text)-len(selText) < 2
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
			if s.clientDecidesScrobble {
				durationEntry.Enable()
			}
			durationEntry.Refresh()
			durationEntry.OnChanged(durationEntry.Text)
		}
	})
	durationEnabled.Checked = s.config.Scrobbling.ThresholdTimeSeconds >= 0
	if !s.config.Scrobbling.Enabled {
		durationEnabled.Disable()
	}
	if !s.clientDecidesScrobble {
		percentEntry.Disable()
		durationEnabled.Disable()
	}

	scrobbleEnabled := widget.NewCheck("Send playback statistics to server", func(checked bool) {
		s.config.Scrobbling.Enabled = checked
		if !checked {
			percentEntry.Disable()
			durationEnabled.Disable()
			durationEntry.Disable()
		} else {
			if s.clientDecidesScrobble {
				percentEntry.Enable()
				durationEnabled.Enable()
			}
			if durationEnabled.Checked && s.clientDecidesScrobble {
				durationEntry.Enable()
			}
		}
	})
	scrobbleEnabled.Checked = s.config.Scrobbling.Enabled

	return container.NewTabItem("General", container.NewVBox(
		container.NewBorder(nil, nil, widget.NewLabel("Theme"), /*left*/
			container.NewHBox(widget.NewLabel("Mode"), themeModeSelect, util.NewHSpace(5)), // right
			themeFileSelect, // center
		),
		container.NewHBox(
			widget.NewLabel("Startup page"), container.NewGridWithColumns(2, startupPage),
		),
		container.NewHBox(systemTrayEnable, closeToTray),
		container.NewHBox(saveQueue),
		container.NewHBox(trackNotif),
		s.newSectionSeparator(),

		widget.NewRichText(&widget.TextSegment{Text: "Scrobbling", Style: util.BoldRichTextStyle}),
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

func (s *SettingsDialog) createPlaybackTab(isLocalPlayer, isReplayGainPlayer bool) *container.TabItem {
	disableTranscode := widget.NewCheckWithData("Disable server transcoding", binding.BindBool(&s.config.Transcoding.ForceRawFile))
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

	replayGainSelect := widget.NewSelect([]string{"None", "Album", "Track", "Auto"}, nil)
	replayGainSelect.OnChanged = func(_ string) {
		switch replayGainSelect.SelectedIndex() {
		case 0:
			s.config.ReplayGain.Mode = backend.ReplayGainNone
		case 1:
			s.config.ReplayGain.Mode = backend.ReplayGainAlbum
		case 2:
			s.config.ReplayGain.Mode = backend.ReplayGainTrack
		case 3:
			s.config.ReplayGain.Mode = backend.ReplayGainAuto
		}
		s.onReplayGainSettingsChanged()
	}

	// set initially selected option
	switch s.config.ReplayGain.Mode {
	case backend.ReplayGainAlbum:
		replayGainSelect.SetSelectedIndex(1)
	case backend.ReplayGainTrack:
		replayGainSelect.SetSelectedIndex(2)
	case backend.ReplayGainAuto:
		replayGainSelect.SetSelectedIndex(3)
	default:
		replayGainSelect.SetSelectedIndex(0)
	}

	preampGain := widgets.NewTextRestrictedEntry(func(curText, _ string, r rune) bool {
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

	if !isLocalPlayer {
		deviceSelect.Disable()
		audioExclusive.Disable()
	}
	if !isReplayGainPlayer {
		replayGainSelect.Disable()
		preventClipping.Disable()
		preampGain.Disable()
	}

	return container.NewTabItem("Playback", container.NewVBox(
		container.NewHBox(disableTranscode),
		container.New(&layouts.MaxPadLayout{PadTop: 5},
			container.New(layout.NewFormLayout(),
				widget.NewLabel("Audio device"), container.NewBorder(nil, nil, nil, util.NewHSpace(70), deviceSelect),
				layout.NewSpacer(), container.NewHBox(audioExclusive, layout.NewSpacer()),
			)),
		s.newSectionSeparator(),

		widget.NewRichText(&widget.TextSegment{Text: "ReplayGain", Style: util.BoldRichTextStyle}),
		container.New(layout.NewFormLayout(),
			widget.NewLabel("ReplayGain mode"), container.NewGridWithColumns(2, replayGainSelect),
			widget.NewLabel("ReplayGain preamp"), container.NewHBox(preampGain, widget.NewLabel("dB")),
			widget.NewLabel("Prevent clipping"), container.NewHBox(preventClipping, layout.NewSpacer()),
		),
	))
}

func (s *SettingsDialog) createEqualizerTab(eqBands []string) *container.TabItem {
	enabled := widget.NewCheck("Enabled", func(b bool) {
		s.config.LocalPlayback.EqualizerEnabled = b
		if s.OnEqualizerSettingsChanged != nil {
			s.OnEqualizerSettingsChanged()
		}
	})
	enabled.Checked = s.config.LocalPlayback.EqualizerEnabled
	geq := NewGraphicEqualizer(s.config.LocalPlayback.EqualizerPreamp,
		eqBands,
		s.config.LocalPlayback.GraphicEqualizerBands)
	debouncer := util.NewDebouncer(350*time.Millisecond, func() {
		if s.OnEqualizerSettingsChanged != nil {
			s.OnEqualizerSettingsChanged()
		}
	})
	geq.OnChanged = func(b int, g float64) {
		s.config.LocalPlayback.GraphicEqualizerBands[b] = g
		debouncer()
	}
	geq.OnPreampChanged = func(g float64) {
		s.config.LocalPlayback.EqualizerPreamp = g
		debouncer()
	}
	cont := container.NewBorder(container.NewHBox(enabled), nil, nil, nil, geq)
	return container.NewTabItem("Equalizer", cont)
}

func (s *SettingsDialog) createExperimentalTab(window fyne.Window) *container.TabItem {
	warningLabel := widget.NewLabel("WARNING: these settings are experimental and may " +
		"make the application buggy or increase system resource use. " +
		"They may be removed in future versions.")
	warningLabel.Wrapping = fyne.TextWrapWord

	normalFontEntry := widget.NewEntry()
	normalFontEntry.SetPlaceHolder("path to .ttf or empty to use default")
	normalFontEntry.Text = s.config.Application.FontNormalTTF
	normalFontEntry.Validator = s.ttfPathValidator
	normalFontEntry.OnChanged = func(path string) {
		if normalFontEntry.Validate() == nil {
			s.setRestartRequired()
			s.config.Application.FontNormalTTF = path
		}
	}
	normalFontBrowse := widget.NewButtonWithIcon("", theme.FolderOpenIcon(), func() {
		s.doChooseTTFFile(window, normalFontEntry)
	})

	boldFontEntry := widget.NewEntry()
	boldFontEntry.SetPlaceHolder("path to .ttf or empty to use default")
	boldFontEntry.Text = s.config.Application.FontBoldTTF
	boldFontEntry.Validator = s.ttfPathValidator
	boldFontEntry.OnChanged = func(path string) {
		if boldFontEntry.Validate() == nil {
			s.setRestartRequired()
			s.config.Application.FontBoldTTF = path
		}
	}
	boldFontBrowse := widget.NewButtonWithIcon("", theme.FolderOpenIcon(), func() {
		s.doChooseTTFFile(window, boldFontEntry)
	})

	uiScaleRadio := widget.NewRadioGroup([]string{"Smaller", "Normal", "Larger"}, func(choice string) {
		s.config.Application.UIScaleSize = choice
		s.setRestartRequired()
	})
	uiScaleRadio.Required = true
	uiScaleRadio.Horizontal = true
	if s.config.Application.UIScaleSize == "Smaller" || s.config.Application.UIScaleSize == "Larger" {
		uiScaleRadio.Selected = s.config.Application.UIScaleSize
	} else {
		uiScaleRadio.Selected = "Normal"
	}
	return container.NewTabItem("Experimental", container.NewVBox(
		warningLabel,
		s.newSectionSeparator(),
		widget.NewRichText(&widget.TextSegment{Text: "UI Scaling", Style: util.BoldRichTextStyle}),
		uiScaleRadio,
		s.newSectionSeparator(),
		widget.NewRichText(&widget.TextSegment{Text: "Application Font", Style: util.BoldRichTextStyle}),
		container.New(layout.NewFormLayout(),
			widget.NewLabel("Normal font"), container.NewBorder(nil, nil, nil, normalFontBrowse, normalFontEntry),
			widget.NewLabel("Bold font"), container.NewBorder(nil, nil, nil, boldFontBrowse, boldFontEntry),
		),
	))
}

func (s *SettingsDialog) doChooseTTFFile(window fyne.Window, entry *widget.Entry) {
	callback := func(urirc fyne.URIReadCloser, err error) {
		if err == nil && urirc != nil {
			entry.SetText(urirc.URI().Path())
		}
	}
	dlg := dialog.NewFileOpen(callback, window)
	dlg.SetFilter(&storage.ExtensionFileFilter{Extensions: []string{".ttf"}})
	dlg.Show()
}

func (s *SettingsDialog) ttfPathValidator(path string) error {
	if path == "" {
		return nil
	}
	if !strings.HasSuffix(path, ".ttf") {
		return errors.New("only .ttf fonts supported")
	}
	_, err := os.Stat(path)
	return err
}

func (s *SettingsDialog) setRestartRequired() {
	ts := s.promptText.Segments[0].(*widget.TextSegment)
	if ts.Text != "" {
		return
	}
	ts.Text = "Restart required"
	ts.Style.ColorName = theme.ColorNameError
	s.promptText.Refresh()
}

func (s *SettingsDialog) newSectionSeparator() fyne.CanvasObject {
	return container.New(&layouts.MaxPadLayout{PadLeft: 15, PadRight: 15}, widget.NewSeparator())
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

func (s *SettingsDialog) saveSelectedTab(tabNum int) {
	var tabName string
	switch tabNum {
	case 0:
		tabName = "General"
	case 1:
		tabName = "Playback"
	case 2:
		tabName = "Equalizer"
	case 3:
		tabName = "Experimental"
	}
	s.config.Application.SettingsTab = tabName
}

func (s *SettingsDialog) getActiveTabNumFromConfig() int {
	switch s.config.Application.SettingsTab {
	case "Playback":
		return 1
	case "Equalizer":
		return 2
	case "Experimental":
		return 3
	default:
		return 0
	}
}
