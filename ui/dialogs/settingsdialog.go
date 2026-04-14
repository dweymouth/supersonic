package dialogs

import (
	"errors"
	"fmt"
	"image/color"
	"log"
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
	"fyne.io/fyne/v2/canvas"
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
	OnPauseFadeSettingsChanged     func()
	OnAudioDeviceSettingChanged    func()
	OnThemeSettingChanged          func()
	OnAccentColorChanged           func() // Lightweight callback for instant accent updates
	OnExtractFromCover             func() // Extract accent color from current track cover art
	OnDismiss                      func()
	OnEqualizerSettingsChanged     func()
	OnPageNeedsRefresh             func()
	OnClearCaches                  func()

	config          *backend.Config
	audioDevices    []mpv.AudioDevice
	themeFiles      map[string]string // filename -> displayName
	promptText      *widget.RichText
	eqPresetManager *backend.EQPresetManager
	autoEQManager   *backend.AutoEQManager
	imageManager    util.ImageFetcher
	window          fyne.Window
	toastProvider   ToastProvider

	clientDecidesScrobble bool

	content fyne.CanvasObject
}

type ToastProvider interface {
	ShowSuccessToast(message string)
	ShowErrorToast(message string)
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
	eqPresetMgr *backend.EQPresetManager,
	window fyne.Window,
	autoEQManager *backend.AutoEQManager,
	imageManager util.ImageFetcher,
	toastProvider ToastProvider,
) *SettingsDialog {
	s := &SettingsDialog{
		config:                config,
		audioDevices:          audioDeviceList,
		themeFiles:            themeFileList,
		clientDecidesScrobble: clientDecidesScrobble,
		eqPresetManager:       eqPresetMgr,
		autoEQManager:         autoEQManager,
		imageManager:          imageManager,
		window:                window,
		toastProvider:         toastProvider,
	}
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
	saveQueue := widget.NewCheck(lang.L("Save play queue"), func(save bool) {
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
	albumGridYears := widget.NewCheck(lang.L("Show year in album grid and now playing"), func(b bool) {
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

	pauseFade := widget.NewCheck(lang.L("Fade out on pause"), func(checked bool) {
		s.config.LocalPlayback.PauseFade = checked
		if s.OnPauseFadeSettingsChanged != nil {
			s.OnPauseFadeSettingsChanged()
		}
	})
	pauseFade.Checked = s.config.LocalPlayback.PauseFade

	if !isLocalPlayer {
		deviceSelect.Disable()
		audioExclusive.Disable()
		pauseFade.Disable()
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
		pauseFade,
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
	// Ensure GraphicEqualizerBands matches the expected number of bands
	if len(s.config.LocalPlayback.GraphicEqualizerBands) != len(eqBands) {
		newBands := make([]float64, len(eqBands))
		copy(newBands, s.config.LocalPlayback.GraphicEqualizerBands)
		s.config.LocalPlayback.GraphicEqualizerBands = newBands
	}

	enabled := widget.NewCheck(lang.L("Enabled"), func(b bool) {
		s.config.LocalPlayback.EqualizerEnabled = b
		if s.OnEqualizerSettingsChanged != nil {
			s.OnEqualizerSettingsChanged()
		}
	})
	enabled.Checked = s.config.LocalPlayback.EqualizerEnabled
	geq := NewGraphicEqualizer(s.config.LocalPlayback.EqualizerPreamp,
		eqBands,
		s.config.LocalPlayback.GraphicEqualizerBands,
		s.config.LocalPlayback.EqualizerType,
		s.eqPresetManager,
		s.window,
		s.config.LocalPlayback.ActiveEQPresetName)
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
	geq.OnManualAdjustment = func() {
		// Clear AutoEQ profile when user manually adjusts sliders
		// Preset name persists so Save button can overwrite the loaded preset
		s.config.LocalPlayback.AutoEQProfilePath = ""
		s.config.LocalPlayback.AutoEQProfileName = ""
		geq.ClearProfileLabel()
	}
	geq.OnLoadAutoEQProfile = func() {
		s.openAutoEQBrowser(geq, debouncer)
	}
	geq.OnPresetSelected = func(presetName string) {
		// Save the active preset name in config
		s.config.LocalPlayback.ActiveEQPresetName = presetName
	}
	geq.OnPresetDeleted = func(presetName string) {
		// Clear active preset name if the deleted preset was active
		if s.config.LocalPlayback.ActiveEQPresetName == presetName {
			s.config.LocalPlayback.ActiveEQPresetName = ""
		}
	}
	geq.OnEQTypeChanged = func(eqType string) {
		// Update config with new EQ type
		s.config.LocalPlayback.EqualizerType = eqType

		// Convert bands using interpolation to preserve EQ curve shape
		var newBands []float64
		currentBands := s.config.LocalPlayback.GraphicEqualizerBands

		if eqType == "ISO10Band" {
			// Converting from 15-band to 10-band
			if len(currentBands) == 15 {
				// Use interpolation to downsample
				var bands15 [15]float64
				copy(bands15[:], currentBands)
				bands10 := backend.InterpolateEQ15BandTo10Band(bands15)
				newBands = bands10[:]
			} else {
				// Already 10-band or invalid size, just copy what we can
				newBands = make([]float64, 10)
				numCopy := min(len(currentBands), 10)
				copy(newBands, currentBands[:numCopy])
			}
		} else {
			// Converting from 10-band to 15-band
			if len(currentBands) == 10 {
				// Use interpolation to upsample
				var bands10 [10]float64
				copy(bands10[:], currentBands)
				bands15 := backend.InterpolateEQ10To15Band(bands10)
				newBands = bands15[:]
			} else {
				// Already 15-band or invalid size, just copy what we can
				newBands = make([]float64, 15)
				numCopy := min(len(currentBands), 15)
				copy(newBands, currentBands[:numCopy])
			}
		}
		s.config.LocalPlayback.GraphicEqualizerBands = newBands

		// Dynamically rebuild the UI with the correct number of sliders
		geq.RebuildForEQType(eqType, newBands)

		// Apply the change to the player
		if s.OnEqualizerSettingsChanged != nil {
			s.OnEqualizerSettingsChanged()
		}
	}

	// Restore profile label if a profile is currently applied
	if s.config.LocalPlayback.AutoEQProfileName != "" {
		geq.SetProfileLabel(s.config.LocalPlayback.AutoEQProfileName)
	}

	cont := container.NewBorder(enabled, nil, nil, nil, geq)
	return container.NewTabItem(lang.L("Equalizer"), cont)
}

func (s *SettingsDialog) openAutoEQBrowser(geq *GraphicEqualizer, debouncer func()) {
	if s.autoEQManager == nil {
		log.Printf("ERROR: AutoEQ manager not available (nil)")
		return
	}
	if s.imageManager == nil {
		log.Printf("ERROR: Image manager not available (nil)")
		return
	}

	browser := NewAutoEQBrowser(s.autoEQManager, s.imageManager, s.toastProvider)

	// Show in a modal popup dialog
	var popup *widget.PopUp
	popup = widget.NewModalPopUp(browser.SearchDialog, s.window.Canvas())

	browser.SetOnProfileSelected(func(profile *backend.AutoEQProfile) {
		s.applyAutoEQProfile(profile, geq, debouncer)
		popup.Hide()
	})
	browser.SetOnDismiss(func() {
		popup.Hide()
	})

	popup.Show()
	s.window.Canvas().Focus(browser.GetSearchEntry())
}

func (s *SettingsDialog) applyAutoEQProfile(profile *backend.AutoEQProfile, geq *GraphicEqualizer, debouncer func()) {
	// Use native 10-band AutoEQ profile
	// Update config to use ISO10Band type
	s.config.LocalPlayback.EqualizerType = "ISO10Band"
	s.config.LocalPlayback.EqualizerPreamp = profile.Preamp
	s.config.LocalPlayback.AutoEQProfilePath = profile.Path
	s.config.LocalPlayback.AutoEQProfileName = profile.Name
	s.config.LocalPlayback.ActiveEQPresetName = "" // Clear preset when applying AutoEQ

	// Ensure GraphicEqualizerBands has the right size for 10 bands
	if len(s.config.LocalPlayback.GraphicEqualizerBands) != 10 {
		s.config.LocalPlayback.GraphicEqualizerBands = make([]float64, 10)
	}

	// Copy native 10-band values
	for i := 0; i < 10; i++ {
		s.config.LocalPlayback.GraphicEqualizerBands[i] = profile.Bands[i]
	}

	// Update UI using applyPreset to avoid triggering manual adjustment
	preset := backend.EQPreset{
		Name:   profile.Name,
		Type:   "ISO10Band",
		Preamp: profile.Preamp,
		Bands:  profile.Bands[:],
	}
	geq.applyPreset(preset)

	// Clear preset dropdown and loaded preset state since AutoEQ is now active
	geq.ClearPresetSelection()
	geq.ClearLoadedPresetState()

	// Show profile label
	geq.SetProfileLabel(profile.Name)

	// Trigger equalizer update
	debouncer()
}

func (s *SettingsDialog) createAppearanceTab(window fyne.Window) *container.TabItem {
	// Theme list: Default, Dynamic, then .toml files
	themeNames := []string{"Default", "Dynamic"}
	themeFileNames := []string{"", "dynamic"}
	i, selIndex := 2, 0

	// Check for Dynamic theme first
	if s.config.Theme.ThemeFile == "dynamic" {
		selIndex = 1
	}

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

	// Mode select - always Dark/Light/Auto for both traditional and dynamic themes
	themeModeSelect := widget.NewSelect([]string{
		string(myTheme.AppearanceDark),
		string(myTheme.AppearanceLight),
		string(myTheme.AppearanceAuto),
	}, nil)
	themeModeSelect.SetSelected(s.config.Theme.Appearance)
	if themeModeSelect.Selected == "" {
		themeModeSelect.SetSelectedIndex(0)
	}

	// Helper to update baseMode from appearance when in dynamic mode
	updateBaseMode := func(appearance string) {
		switch appearance {
		case string(myTheme.AppearanceDark):
			s.config.Theme.BaseMode = "black"
		case string(myTheme.AppearanceLight):
			s.config.Theme.BaseMode = "light"
		case string(myTheme.AppearanceAuto):
			// Auto uses grey as default (will adapt to system in future)
			s.config.Theme.BaseMode = "grey"
		}
	}

	themeFileSelect.OnChanged = func(_ string) {
		s.config.Theme.ThemeFile = themeFileNames[themeFileSelect.SelectedIndex()]
		// Update base mode if switching to dynamic
		if s.config.Theme.ThemeFile == "dynamic" {
			updateBaseMode(themeModeSelect.Selected)
		}
		if s.OnThemeSettingChanged != nil {
			s.OnThemeSettingChanged()
		}
	}

	themeModeSelect.OnChanged = func(value string) {
		s.config.Theme.Appearance = value
		// If in dynamic mode, also update baseMode
		if s.config.Theme.ThemeFile == "dynamic" {
			updateBaseMode(value)
		}
		if s.OnThemeSettingChanged != nil {
			s.OnThemeSettingChanged()
		}
	}

	// Set initial base mode if dynamic is selected
	if s.config.Theme.ThemeFile == "dynamic" {
		updateBaseMode(s.config.Theme.Appearance)
	}

	// Ensure we have valid values (defaults should be set in config loading, but ensure here)
	// Also update the config with defaults so they get saved
	if s.config.Theme.AccentColor == "" {
		s.config.Theme.AccentColor = "#FF8A45" // Default orange accent
	}
	if s.config.Theme.Saturation == 0 {
		s.config.Theme.Saturation = 1.0
	}
	if s.config.Theme.Contrast == 0 {
		s.config.Theme.Contrast = 1.0
	}
	if s.config.Theme.BaseMode == "" || s.config.Theme.BaseMode == "dark" {
		s.config.Theme.BaseMode = "grey"
	}

	// Hue slider (0-360) for rainbow color selection
	hueSlider := widget.NewSlider(0, 360)
	hueSlider.SetValue(hexToHue(s.config.Theme.AccentColor))
	hueSlider.Step = 1

	// Color preview rectangle
	colorPreview := canvas.NewRectangle(colorFromHex(s.config.Theme.AccentColor))
	colorPreview.SetMinSize(fyne.NewSize(40, 24))

	hueSlider.OnChanged = func(hue float64) {
		s.config.Theme.AccentColor = hueToHex(hue)
		colorPreview.FillColor = colorFromHex(s.config.Theme.AccentColor)
		colorPreview.Refresh()
		if s.OnAccentColorChanged != nil {
			s.OnAccentColorChanged()
		}
	}

	// Helper to get ranges from theme package (single source of truth)
	getRanges := func(mode string) myTheme.SliderRanges {
		return myTheme.GetSliderRanges(mode)
	}
	clamp := func(val, min, max float64) float64 {
		if val < min {
			return min
		}
		if val > max {
			return max
		}
		return val
	}

	// Create sliders with initial ranges based on current base mode
	initRanges := getRanges(s.config.Theme.BaseMode)

	saturationSlider := widget.NewSlider(initRanges.SatMin, initRanges.SatMax)
	saturationSlider.SetValue(clamp(s.config.Theme.Saturation, initRanges.SatMin, initRanges.SatMax))
	saturationSlider.Step = 0.05

	contrastSlider := widget.NewSlider(initRanges.ContrastMin, initRanges.ContrastMax)
	contrastSlider.SetValue(clamp(s.config.Theme.Contrast, initRanges.ContrastMin, initRanges.ContrastMax))
	contrastSlider.Step = 0.05

	// Set up OnChanged handlers for sliders
	saturationSlider.OnChanged = func(f float64) {
		s.config.Theme.Saturation = f
		if s.OnAccentColorChanged != nil {
			s.OnAccentColorChanged()
		}
	}

	contrastSlider.OnChanged = func(f float64) {
		s.config.Theme.Contrast = f
		if s.OnAccentColorChanged != nil {
			s.OnAccentColorChanged()
		}
	}

	// Helper to check if Dynamic theme is selected
	isDynamic := func() bool {
		return themeFileNames[themeFileSelect.SelectedIndex()] == "dynamic"
	}

	// Enable/disable accent controls based on Dynamic theme selection
	updateAccentControls := func() {
		if isDynamic() {
			hueSlider.Enable()
			saturationSlider.Enable()
			contrastSlider.Enable()
		} else {
			hueSlider.Disable()
			saturationSlider.Disable()
			contrastSlider.Disable()
		}
	}

	// Initial state
	updateAccentControls()

	// Update controls when theme changes
	themeFileSelect.OnChanged = func(name string) {
		s.config.Theme.ThemeFile = themeFileNames[themeFileSelect.SelectedIndex()]
		// Update base mode if switching to dynamic
		if s.config.Theme.ThemeFile == "dynamic" {
			updateBaseMode(themeModeSelect.Selected)
		}
		updateAccentControls()
		if s.OnThemeSettingChanged != nil {
			s.OnThemeSettingChanged()
		}
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

	nowPlayingBackground := widget.NewCheckWithData(lang.L("Use blurred album cover for Now Playing page background"), binding.BindBool(&s.config.NowPlayingConfig.UseBackgroundImage))

	useRoundedImageCorners := widget.NewCheck(lang.L("Use rounded image corners"), func(b bool) {
		s.config.Theme.UseRoundedImageCorners = b
		if s.OnPageNeedsRefresh != nil {
			s.OnPageNeedsRefresh()
		}
	})
	useRoundedImageCorners.Checked = s.config.Theme.UseRoundedImageCorners

	// Create accent color controls container (shown only for Dynamic theme)
	accentControls := container.NewVBox(
		s.newSectionSeparator(),
		widget.NewRichText(&widget.TextSegment{Text: lang.L("Accent Color"), Style: util.BoldRichTextStyle}),
		container.NewBorder(nil, nil, nil, colorPreview, hueSlider),
		container.New(layout.NewFormLayout(),
			widget.NewLabel(lang.L("Saturation")), saturationSlider,
			widget.NewLabel(lang.L("Contrast")), contrastSlider,
		),
		s.newSectionSeparator(),
		widget.NewRichText(&widget.TextSegment{Text: lang.L("Dynamic Accent from Cover Art"), Style: util.BoldRichTextStyle}),
		container.NewHBox(
			widget.NewButton(lang.L("Extract from playing track"), func() {
				if s.OnExtractFromCover != nil {
					s.OnExtractFromCover()
				}
			}),
			util.NewHSpace(10),
		),
	)

	// Show/hide accent controls based on Dynamic theme selection
	updateAccentVisibility := func() {
		if isDynamic() {
			accentControls.Show()
		} else {
			accentControls.Hide()
		}
		accentControls.Refresh()
	}

	// Initial visibility
	updateAccentVisibility()

	// Update visibility when theme changes
	originalThemeFileOnChanged := themeFileSelect.OnChanged
	themeFileSelect.OnChanged = func(name string) {
		if originalThemeFileOnChanged != nil {
			originalThemeFileOnChanged(name)
		}
		updateAccentVisibility()
	}

	return container.NewTabItem(lang.L("Appearance"), container.NewVBox(
		util.NewHSpace(0), // insert a theme.Padding amount of space at top
		container.NewBorder(nil, nil, widget.NewLabel(lang.L("Theme")), /*left*/
			container.NewHBox(widget.NewLabel(lang.L("Mode")), themeModeSelect, util.NewHSpace(5)), // right
			themeFileSelect, // center
		),
		accentControls,
		widget.NewRichText(&widget.TextSegment{Text: lang.L("UI Scaling"), Style: util.BoldRichTextStyle}),
		uiScaleRadio,
		container.NewBorder(nil, nil, widget.NewLabel(lang.L("Grid card size")), nil, gridCardSize),
		disableDPI,
		s.newSectionSeparator(),
		useWaveformSeekbar,
		nowPlayingBackground,
		useRoundedImageCorners,
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

	clearCaches := widget.NewButton(lang.L("Clear caches"), func() {
		if s.OnClearCaches != nil {
			s.OnClearCaches()
		}
	})

	imgCacheCfg := container.NewHBox(
		widget.NewLabel(lang.L("Maximum image cache size")),
		percentEntry,
		widget.NewLabel("MB"),
		layout.NewSpacer(),
		clearCaches,
	)

	osMediaAPIs := widget.NewCheck(lang.L("Enable OS media player integration"), func(b bool) {
		s.config.Application.EnableOSMediaPlayerAPIs = b
		s.setRestartRequired()
	})
	osMediaAPIs.Checked = s.config.Application.EnableOSMediaPlayerAPIs

	preventScreensaver := widget.NewCheckWithData(lang.L("Prevent screensaver on Now Playing page"),
		binding.BindBool(&s.config.Application.PreventScreensaverOnNowPlayingPage))

	return container.NewTabItem(lang.L("Advanced"), container.NewVBox(
		multi,
		update,
		lrclib,
		osMediaAPIs,
		preventScreensaver,
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

// colorFromHex converts a hex color string to a color.Color
func colorFromHex(hex string) color.Color {
	r, g, b := hexToRGB(hex)
	return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}
}

// hexToRGB converts a hex color string to RGB values
func hexToRGB(hex string) (r, g, b float64) {
	if len(hex) < 7 || hex[0] != '#' {
		return 255, 136, 69 // default orange
	}
	rval, _ := strconv.ParseInt(hex[1:3], 16, 64)
	gval, _ := strconv.ParseInt(hex[3:5], 16, 64)
	bval, _ := strconv.ParseInt(hex[5:7], 16, 64)
	return float64(rval), float64(gval), float64(bval)
}

// rgbToHex converts RGB values to a hex color string
func rgbToHex(r, g, b float64) string {
	return fmt.Sprintf("#%02X%02X%02X", int(r), int(g), int(b))
}

// hexToHue extracts the hue (0-360) from a hex color
func hexToHue(hex string) float64 {
	r, g, b := hexToRGB(hex)
	return rgbToHue(r, g, b)
}

// rgbToHue converts RGB to hue value (0-360)
func rgbToHue(r, g, b float64) float64 {
	max := math.Max(r, math.Max(g, b))
	min := math.Min(r, math.Min(g, b))
	delta := max - min

	if delta == 0 {
		return 0
	}

	var hue float64
	switch {
	case max == r:
		hue = ((g - b) / delta)
		if g < b {
			hue += 6
		}
	case max == g:
		hue = ((b-r)/delta + 2)
	default:
		hue = ((r-g)/delta + 4)
	}

	return hue * 60
}

// hueToRGB converts a hue value (0-360) to a full saturation RGB color
func hueToRGB(hue float64) (r, g, b float64) {
	hue = math.Mod(hue, 360)
	if hue < 0 {
		hue += 360
	}

	c := 255.0 // full saturation
	x := c * (1 - math.Abs(math.Mod(hue/60, 2)-1))
	m := 0.0

	var r1, g1, b1 float64
	switch {
	case hue < 60:
		r1, g1, b1 = c, x, 0
	case hue < 120:
		r1, g1, b1 = x, c, 0
	case hue < 180:
		r1, g1, b1 = 0, c, x
	case hue < 240:
		r1, g1, b1 = 0, x, c
	case hue < 300:
		r1, g1, b1 = x, 0, c
	default:
		r1, g1, b1 = c, 0, x
	}

	return r1 + m, g1 + m, b1 + m
}

// hueToHex converts a hue value (0-360) to a hex color string
func hueToHex(hue float64) string {
	r, g, b := hueToRGB(hue)
	return rgbToHex(r, g, b)
}
