package dialogs

import (
	"fmt"

	"github.com/dweymouth/supersonic/backend"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type AddEditServerDialog struct {
	widget.BaseWidget

	ServerType       backend.ServerType
	Nickname         string
	Host             string
	AltHost          string
	Username         string
	Password         string
	LegacyAuth       bool
	SkipSSLVerify    bool
	StopOnDisconnect bool
	OnSubmit         func()
	OnCancel         func()

	passField            *widget.Entry
	submitBtn            *widget.Button
	promptText           *widget.RichText
	stopOnDisconnectChk  *widget.Check
	container            *fyne.Container
}

var _ fyne.Widget = (*AddEditServerDialog)(nil)

func NewAddEditServerDialog(title string, cancelable bool, prefillServer *backend.ServerConfig, focusHandler func(fyne.Focusable)) *AddEditServerDialog {
	a := &AddEditServerDialog{}
	a.ExtendBaseWidget(a)
	if prefillServer != nil {
		a.ServerType = prefillServer.ServerType
		a.Nickname = prefillServer.Nickname
		a.Host = prefillServer.Hostname
		a.AltHost = prefillServer.AltHostname
		a.Username = prefillServer.Username
		a.LegacyAuth = prefillServer.LegacyAuth
		a.SkipSSLVerify = prefillServer.SkipSSLVerify
		a.StopOnDisconnect = prefillServer.StopOnDisconnect
	}

	titleLabel := widget.NewLabel(title)
	titleLabel.TextStyle.Bold = true
	legacyAuthCheck := widget.NewCheckWithData(lang.L("Use legacy authentication"), binding.BindBool(&a.LegacyAuth))
	a.stopOnDisconnectChk = widget.NewCheckWithData(lang.L("Stop playback on disconnect"), binding.BindBool(&a.StopOnDisconnect))
	var altHostField, userField *widget.Entry
	var hostLabel, altHostLabel, userLabel *widget.Label

	serverTypeChoice := widget.NewRadioGroup([]string{"Subsonic", "Jellyfin", "MPD"}, func(s string) {
		a.ServerType = backend.ServerType(s)
		switch a.ServerType {
		case backend.ServerTypeSubsonic:
			legacyAuthCheck.Show()
			a.stopOnDisconnectChk.Hide()
			if altHostField != nil {
				altHostField.Show()
				altHostLabel.Show()
			}
			if userField != nil {
				userField.Show()
				userLabel.Show()
			}
			if hostLabel != nil {
				hostLabel.SetText(lang.L("URL"))
			}
		case backend.ServerTypeJellyfin:
			legacyAuthCheck.Hide()
			a.stopOnDisconnectChk.Hide()
			if altHostField != nil {
				altHostField.Show()
				altHostLabel.Show()
			}
			if userField != nil {
				userField.Show()
				userLabel.Show()
			}
			if hostLabel != nil {
				hostLabel.SetText(lang.L("URL"))
			}
		case backend.ServerTypeMPD:
			legacyAuthCheck.Hide()
			a.stopOnDisconnectChk.Show()
			if altHostField != nil {
				altHostField.Hide()
				altHostLabel.Hide()
			}
			if userField != nil {
				userField.Hide()
				userLabel.Hide()
			}
			if hostLabel != nil {
				hostLabel.SetText(lang.L("Host:Port"))
			}
		}
	})
	skipSSLCheck := widget.NewCheckWithData(lang.L("Skip SSL certificate verification"), binding.BindBool(&a.SkipSSLVerify))
	serverTypeChoice.Required = true
	serverTypeChoice.Horizontal = true
	selected := backend.ServerTypeSubsonic
	if a.ServerType == backend.ServerTypeJellyfin {
		selected = backend.ServerTypeJellyfin
	} else if a.ServerType == backend.ServerTypeMPD {
		selected = backend.ServerTypeMPD
	}
	serverTypeChoice.Selected = string(selected)
	a.passField = widget.NewPasswordEntry()
	a.passField.OnSubmitted = func(_ string) { a.doSubmit() }
	userField = widget.NewEntryWithData(binding.BindString(&a.Username))
	userField.OnSubmitted = func(_ string) { focusHandler(a.passField) }
	altHostField = widget.NewEntryWithData(binding.BindString(&a.AltHost))
	altHostField.SetPlaceHolder(fmt.Sprintf("(%s)", lang.L("optional")) + " https://my-external-domain.net/music")
	altHostField.OnSubmitted = func(_ string) { focusHandler(userField) }
	hostField := widget.NewEntryWithData(binding.BindString(&a.Host))
	hostField.SetPlaceHolder("http://localhost:4533")
	hostField.OnSubmitted = func(_ string) { focusHandler(altHostField) }

	// Create labels as variables so they can be modified
	hostLabel = widget.NewLabel(lang.L("URL"))
	altHostLabel = widget.NewLabel(lang.L("Alt. URL"))
	userLabel = widget.NewLabel(lang.L("Username"))

	// Set appropriate placeholders based on server type
	if a.ServerType == backend.ServerTypeMPD {
		hostField.SetPlaceHolder("localhost:6600")
	}
	nickField := widget.NewEntryWithData(binding.BindString(&a.Nickname))
	nickField.SetPlaceHolder(lang.L("My Server"))
	nickField.OnSubmitted = func(_ string) { focusHandler(hostField) }
	a.submitBtn = widget.NewButtonWithIcon(lang.L("Enter"), theme.ConfirmIcon(), a.doSubmit)
	a.submitBtn.Importance = widget.HighImportance
	a.promptText = widget.NewRichTextWithText("")
	a.promptText.Hidden = true

	var bottomRow *fyne.Container
	if cancelable {
		bottomRow = container.NewHBox(
			a.promptText,
			layout.NewSpacer(),
			widget.NewButtonWithIcon(lang.L("Cancel"), theme.CancelIcon(), a.onCancel),
			a.submitBtn)
	} else {
		bottomRow = container.NewHBox(
			a.promptText,
			layout.NewSpacer(),
			a.submitBtn)
	}

	a.container = container.NewVBox(
		container.NewHBox(layout.NewSpacer(), titleLabel, layout.NewSpacer()),
		container.New(layout.NewFormLayout(),
			widget.NewLabel(lang.L("Server Type")),
			serverTypeChoice,
			widget.NewLabel(lang.L("Nickname")),
			nickField,
			hostLabel,
			hostField,
			altHostLabel,
			altHostField,
			userLabel,
			userField,
			widget.NewLabel(lang.L("Password")),
			a.passField,
		),
		container.NewHBox(layout.NewSpacer(), legacyAuthCheck, skipSSLCheck),
		container.NewHBox(layout.NewSpacer(), a.stopOnDisconnectChk),
		widget.NewSeparator(),
		bottomRow,
	)

	// Trigger initial visibility based on server type
	serverTypeChoice.SetSelected(string(selected))
	return a
}

func (a *AddEditServerDialog) SetInfoText(text string) {
	a.doSetPromptText(text, theme.ColorNameForeground)
}

func (a *AddEditServerDialog) SetErrorText(text string) {
	a.doSetPromptText(text, theme.ColorNameError)
}

func (a *AddEditServerDialog) EnableSubmit() {
	a.submitBtn.Enable()
	a.submitBtn.Refresh()
}

func (a *AddEditServerDialog) DisableSubmit() {
	a.submitBtn.Disable()
	a.submitBtn.Refresh()
}

func (a *AddEditServerDialog) doSubmit() {
	a.Password = a.passField.Text
	if a.OnSubmit != nil {
		a.OnSubmit()
	}
}

func (a *AddEditServerDialog) onCancel() {
	if a.OnCancel != nil {
		a.OnCancel()
	}
}

func (a *AddEditServerDialog) doSetPromptText(text string, color fyne.ThemeColorName) {
	ts := a.promptText.Segments[0].(*widget.TextSegment)
	ts.Text = text
	ts.Style.ColorName = color
	if text != "" {
		a.promptText.Show()
	} else {
		a.promptText.Hide()
	}
	a.promptText.Refresh()
}

func (a *AddEditServerDialog) MinSize() fyne.Size {
	a.ExtendBaseWidget(a)
	return fyne.NewSize(475, a.container.MinSize().Height)
}

func (a *AddEditServerDialog) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}
