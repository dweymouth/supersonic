package dialogs

import (
	"errors"
	"math"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/backend/player/mpv"
	"github.com/dweymouth/supersonic/res"
	myTheme "github.com/dweymouth/supersonic/ui/theme"
	"github.com/dweymouth/supersonic/ui/util"
	"github.com/dweymouth/supersonic/ui/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"
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
	OnPageNeedsRefresh             func()

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
	canSavePlayQueue bool,
	window fyne.Window,
) *SettingsDialog {
	s := &SettingsDialog{config: config, audioDevices: audioDeviceList, themeFiles: themeFileList, clientDecidesScrobble: clientDecidesScrobble}
	s.ExtendBaseWidget(s)

	// TODO: It may be a nicer UX to always create the equalizer tab,
	// but disable it if we are not using an equalizer player
	var tabs *container.AppTabs
	if isEqualizerPlayer {
		tabs = container.NewAppTabs(
			s.createGeneralTab(canSavePlayQueue),
			s.createAppearanceTab(window),
			s.createPlaybackTab(isLocalPlayer, isReplayGainPlayer),
			s.createEqualizerTab(equalizerBands),
			s.createAdvancedTab(),
		)
	} else {
		tabs = container.NewAppTabs(
			s.createGeneralTab(canSavePlayQueue),
			s.createAppearanceTab(window),
			s.createPlaybackTab(isLocalPlayer, isReplayGainPlayer),
			s.createAdvancedTab(),
		)
	}

	tabs.SelectIndex(s.getActiveTabNumFromConfig())
	tabs.OnSelected = func(ti *container.TabItem) {
		s.saveSelectedTab(tabs.SelectedIndex())
	}
	s.promptText = widget.NewRichTextWithText("")
	s.content = container.NewVBox(tabs, widget.NewSeparator(),
		container.NewHBox(s.promptText, layout.NewSpacer(), widget.NewButton(lang.L("Close"), func() {
			if s.OnDismiss != nil {
				s.OnDismiss()
			}
		})))

	return s
}

func (s *SettingsDialog) createGeneralTab(canSaveQueueToServer bool) *container.TabItem {
	pages := util.LocalizeSlice(backend.SupportedStartupPages)
	var startupPage *widget.Select
	startupPage = widget.NewSelect(pages, func(_ string) {
		s.config.Application.StartupPage = backend.SupportedStartupPages[startupPage.SelectedIndex()]
	})
	initialIdx := max(slices.Index(backend.SupportedStartupPages, s.config.Application.StartupPage), 0)
	startupPage.SetSelectedIndex(initialIdx)
	if startupPage.Selected == "" {
		startupPage.SetSelectedIndex(0)
	}

	languageList := make([]string, len(res.TranslationsInfo)+1)
	languageList[0] = lang.L("Auto")
	var langSelIndex int
	for i, tr := range res.TranslationsInfo {
		languageList[i+1] = tr.DisplayName
		if tr.Name == s.config.Application.Language {
			langSelIndex = i + 1
		}
	}

	languageSelect := widget.NewSelect(languageList, nil)
	languageSelect.SetSelectedIndex(langSelIndex)
	languageSelect.OnChanged = func(_ string) {
		lang := "auto"
		if i := languageSelect.SelectedIndex(); i > 0 {
			lang = res.TranslationsInfo[i-1].Name
		}
		s.config.Application.Language = lang
		s.setRestartRequired()
	}

	closeToTray := widget.NewCheckWithData(lang.L("Close to system tray"),
		binding.BindBool(&s.config.Application.CloseToSystemTray))
	if !s.config.Application.EnableSystemTray {
		closeToTray.Disable()
	}
	systemTrayEnable := widget.NewCheck(lang.L("Enable system tray"), func(val bool) {
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

	// save play queue settings
	locally := lang.L("Locally")
	toServer := lang.L("To server")
	saveToServer := widget.NewRadioGroup([]string{locally, toServer}, func(choice string) {
		s.config.Application.SaveQueueToServer = choice == toServer
	})
	saveToServer.Horizontal = true
	if !s.config.Application.SavePlayQueue {
		saveToServer.Disable()
	}
	saveToServer.Selected = locally
	if s.config.Application.SaveQueueToServer {
		saveToServer.Selected = toServer
	}
	saveQueue := widget.NewCheck(lang.L("Save play queue on exit"), func(save bool) {
		s.config.Application.SavePlayQueue = save
		if save && canSaveQueueToServer {
			saveToServer.Enable()
		} else if canSaveQueueToServer {
			saveToServer.Disable()
		}
	})
	saveQueue.Checked = s.config.Application.SavePlayQueue
	saveQueueHBox := container.NewHBox(saveQueue)
	if canSaveQueueToServer {
		saveQueueHBox.Add(saveToServer)
	}

	trackNotif := widget.NewCheckWithData(lang.L("Show notification on track change"),
		binding.BindBool(&s.config.Application.ShowTrackChangeNotification))
	albumGridYears := widget.NewCheck(lang.L("Show year in album grid cards"), func(b bool) {
		s.config.AlbumsPage.ShowYears = b
		s.config.FavoritesPage.ShowAlbumYears = b
		if s.OnPageNeedsRefresh != nil {
			s.OnPageNeedsRefresh()
		}
	})
	albumGridYears.Checked = s.config.AlbumsPage.ShowYears

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
	durationEnabled := widget.NewCheck(lang.L("or when"), func(checked bool) {
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

	scrobbleEnabled := widget.NewCheck(lang.L("Send playback statistics to server"), func(checked bool) {
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

	return container.NewTabItem(lang.L("General"), container.NewVBox(
		util.NewHSpace(0), // insert a theme.Padding amount of space at top
		container.NewHBox(widget.NewLabel(lang.L("Language")), languageSelect),
		container.NewHBox(
			widget.NewLabel(lang.L("Startup page")), container.NewGridWithColumns(2, startupPage),
		),
		container.NewHBox(systemTrayEnable, closeToTray),
		saveQueueHBox,
		trackNotif,
		albumGridYears,
		s.newSectionSeparator(),

		widget.NewRichText(&widget.TextSegment{Text: "Scrobbling", Style: util.BoldRichTextStyle}),
		scrobbleEnabled,
		container.NewHBox(
			widget.NewLabel(lang.L("Scrobble when")),
			percentEntry,
			widget.NewLabel(lang.L("percent of track is played")),
		),
		container.NewHBox(
			durationEnabled,
			durationEntry,
			widget.NewLabel(lang.L("minutes of track have been played")),
		),
	))
}

func (s *SettingsDialog) createPlaybackTab(isLocalPlayer, isReplayGainPlayer bool) *container.TabItem {
	transcodeCodec := widget.NewSelectWithData([]string{"opus", "mp3"}, binding.BindString(&s.config.Transcoding.Codec))
	transcodeBitRate := widget.NewSelectWithData([]string{"96", "128", "160", "192", "256", "320"},
		binding.IntToString(binding.BindInt(&s.config.Transcoding.MaxBitRateKBPS)))
	if !s.config.Transcoding.RequestTranscode {
		transcodeCodec.Disable()
		transcodeBitRate.Disable()
	}

	var transcode *widget.Check
	disableTranscode := widget.NewCheck(lang.L("Disable server transcoding"), func(b bool) {
		s.config.Transcoding.ForceRawFile = b
		if b {
			transcode.SetChecked(false)
		}
	})
	disableTranscode.Checked = s.config.Transcoding.ForceRawFile
	transcode = widget.NewCheck(lang.L("Transcode to"), func(b bool) {
		s.config.Transcoding.RequestTranscode = b
		if b {
			disableTranscode.SetChecked(false)
			transcodeCodec.Enable()
			transcodeBitRate.Enable()
		} else {
			transcodeCodec.Disable()
			transcodeBitRate.Disable()
		}
	})
	transcode.Checked = s.config.Transcoding.RequestTranscode

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

	rGainOpts := []string{lang.L("None"), lang.L("Album"), lang.L("Track"), lang.L("Auto")}
	replayGainSelect := widget.NewSelect(rGainOpts, nil)
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

	audioExclusive := widget.NewCheck(lang.L("Exclusive mode"), func(checked bool) {
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

	return container.NewTabItem(lang.L("Playback"), container.NewVBox(

		container.New(&layout.CustomPaddedLayout{TopPadding: 5},
			container.New(layout.NewFormLayout(),
				widget.NewLabel(lang.L("Audio device")), container.NewBorder(nil, nil, nil, util.NewHSpace(70), deviceSelect),
				layout.NewSpacer(), audioExclusive,
			)),
		s.newSectionSeparator(),
		disableTranscode,
		container.NewHBox(transcode, transcodeCodec, transcodeBitRate),
		s.newSectionSeparator(),
		widget.NewLabelWithStyle("ReplayGain", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.New(layout.NewFormLayout(),
			widget.NewLabel(lang.L("ReplayGain mode")), container.NewGridWithColumns(2, replayGainSelect),
			widget.NewLabel(lang.L("ReplayGain preamp")), container.NewHBox(preampGain, widget.NewLabel("dB")),
			widget.NewLabel(lang.L("Prevent clipping")), preventClipping,
		),
		s.newSectionSeparator(),
		widget.NewLabelWithStyle(lang.L("When enqueuing random"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewCheckWithData(lang.L("Skip one-star tracks"), binding.BindBool(&s.config.Playback.SkipOneStarWhenShuffling)),
		container.NewBorder(nil, nil,
			widget.NewLabel(lang.L("Skip tracks with keyword")), nil,
			widget.NewEntryWithData(binding.BindString(&s.config.Playback.SkipKeywordWhenShuffling)),
		),
	))
}

func (s *SettingsDialog) createEqualizerTab(eqBands []string) *container.TabItem {
	enabled := widget.NewCheck(lang.L("Enabled"), func(b bool) {
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
	cont := container.NewBorder(enabled, nil, nil, nil, geq)
	return container.NewTabItem(lang.L("Equalizer"), cont)
}

func (s *SettingsDialog) createAppearanceTab(window fyne.Window) *container.TabItem {
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
		string(myTheme.AppearanceAuto),
	}, nil)
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

	uiScaleRadio := widget.NewRadioGroup([]string{lang.L("Smaller"), lang.L("Normal"), lang.L("Larger")}, func(choice string) {
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

	disableDPI := widget.NewCheck(lang.L("Disable automatic DPI adjustment"), func(b bool) {
		s.config.Application.DisableDPIDetection = b
		s.setRestartRequired()
	})
	disableDPI.Checked = s.config.Application.DisableDPIDetection

	gridCardSize := widget.NewSlider(150, 350)
	gridCardSize.SetValue(float64(s.config.GridView.CardSize))
	gridCardSize.Step = 10
	gridCardSize.OnChanged = func(f float64) {
		s.config.GridView.CardSize = float32(f)
		if s.OnPageNeedsRefresh != nil {
			s.OnPageNeedsRefresh()
		}
	}

	useWaveformSeekbar := widget.NewCheck(lang.L("Use waveform seekbar"), func(b bool) {
		s.config.Playback.UseWaveformSeekbar = b
		s.setRestartRequired()
	})
	useWaveformSeekbar.Checked = s.config.Playback.UseWaveformSeekbar

	return container.NewTabItem(lang.L("Appearance"), container.NewVBox(
		util.NewHSpace(0), // insert a theme.Padding amount of space at top
		container.NewBorder(nil, nil, widget.NewLabel(lang.L("Theme")), /*left*/
			container.NewHBox(widget.NewLabel(lang.L("Mode")), themeModeSelect, util.NewHSpace(5)), // right
			themeFileSelect, // center
		),
		widget.NewRichText(&widget.TextSegment{Text: lang.L("UI Scaling"), Style: util.BoldRichTextStyle}),
		uiScaleRadio,
		container.NewBorder(nil, nil, widget.NewLabel(lang.L("Grid card size")), nil, gridCardSize),
		disableDPI,
		s.newSectionSeparator(),
		useWaveformSeekbar,
		s.newSectionSeparator(),
		widget.NewRichText(&widget.TextSegment{Text: lang.L("Application font"), Style: util.BoldRichTextStyle}),
		container.New(layout.NewFormLayout(),
			widget.NewLabel(lang.L("Normal font")), container.NewBorder(nil, nil, nil, normalFontBrowse, normalFontEntry),
			widget.NewLabel(lang.L("Bold font")), container.NewBorder(nil, nil, nil, boldFontBrowse, boldFontEntry),
		),
	))
}

func (s *SettingsDialog) createAdvancedTab() *container.TabItem {
	multi := widget.NewCheckWithData(lang.L("Allow multiple app instances"), binding.BindBool(&s.config.Application.AllowMultiInstance))
	sslSkip := widget.NewCheckWithData(lang.L("Skip SSL certificate verification"), binding.BindBool(&s.config.Application.SkipSSLVerify))
	update := widget.NewCheckWithData(lang.L("Automatically check for updates"), binding.BindBool(&s.config.Application.EnableAutoUpdateChecker))
	lrclib := widget.NewCheckWithData(lang.L("Enable LrcLib lyrics fetcher"), binding.BindBool(&s.config.Application.EnableLrcLib))

	threeDigitValidator := func(text, selText string, r rune) bool {
		return unicode.IsDigit(r) && len(text)-len(selText) < 3
	}

	percentEntry := widgets.NewTextRestrictedEntry(threeDigitValidator)
	percentEntry.SetMinCharWidth(3)
	percentEntry.OnChanged = func(str string) {
		if i, err := strconv.Atoi(str); err == nil {
			s.config.Application.MaxImageCacheSizeMB = i
		}
	}
	percentEntry.Text = strconv.Itoa(s.config.Application.MaxImageCacheSizeMB)

	imgCacheCfg := container.NewHBox(
		widget.NewLabel(lang.L("Maximum image cache size")),
		percentEntry,
		widget.NewLabel("MB"),
	)

	osMediaAPIs := widget.NewCheck(lang.L("Enable OS media player integration"), func(b bool) {
		s.config.Application.EnableOSMediaPlayerAPIs = b
		s.setRestartRequired()
	})
	osMediaAPIs.Checked = s.config.Application.EnableOSMediaPlayerAPIs

	return container.NewTabItem(lang.L("Advanced"), container.NewVBox(
		multi,
		sslSkip,
		update,
		lrclib,
		osMediaAPIs,
		imgCacheCfg,
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
	ts.Text = lang.L("Restart required")
	ts.Style.ColorName = theme.ColorNameError
	s.promptText.Refresh()
}

func (s *SettingsDialog) newSectionSeparator() fyne.CanvasObject {
	return container.New(&layout.CustomPaddedLayout{LeftPadding: 15, RightPadding: 15}, widget.NewSeparator())
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
