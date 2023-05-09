package dialogs

import (
	"github.com/dweymouth/supersonic/backend"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type AddEditServerDialog struct {
	widget.BaseWidget

	Nickname   string
	Host       string
	AltHost    string
	Username   string
	Password   string
	LegacyAuth bool
	OnSubmit   func()

	passField  *widget.Entry
	submitBtn  *widget.Button
	promptText *widget.RichText
	container  *fyne.Container
}

var _ fyne.Widget = (*AddEditServerDialog)(nil)

func NewAddEditServerDialog(title string, prefillServer *backend.ServerConfig, focusHandler func(fyne.Focusable)) *AddEditServerDialog {
	a := &AddEditServerDialog{}
	a.ExtendBaseWidget(a)
	if prefillServer != nil {
		a.Nickname = prefillServer.Nickname
		a.Host = prefillServer.Hostname
		a.AltHost = prefillServer.AltHostname
		a.Username = prefillServer.Username
		a.LegacyAuth = prefillServer.LegacyAuth
	}

	titleLabel := widget.NewLabel(title)
	titleLabel.TextStyle.Bold = true
	a.passField = widget.NewPasswordEntry()
	a.passField.OnSubmitted = func(_ string) { a.doSubmit() }
	userField := widget.NewEntryWithData(binding.BindString(&a.Username))
	userField.OnSubmitted = func(_ string) { focusHandler(a.passField) }
	altHostField := widget.NewEntryWithData(binding.BindString(&a.AltHost))
	altHostField.SetPlaceHolder("(optional) https://my-external-domain.net/music")
	altHostField.OnSubmitted = func(_ string) { focusHandler(userField) }
	hostField := widget.NewEntryWithData(binding.BindString(&a.Host))
	hostField.SetPlaceHolder("http://localhost:4533")
	hostField.OnSubmitted = func(_ string) { focusHandler(altHostField) }
	nickField := widget.NewEntryWithData(binding.BindString(&a.Nickname))
	nickField.SetPlaceHolder("My Server")
	nickField.OnSubmitted = func(_ string) { focusHandler(hostField) }
	a.submitBtn = widget.NewButton("Enter", a.doSubmit)
	a.promptText = widget.NewRichTextWithText("")
	a.promptText.Hidden = true

	legacyAuthCheck := widget.NewCheckWithData("Use legacy authentication", binding.BindBool(&a.LegacyAuth))

	a.container = container.NewVBox(
		container.NewHBox(layout.NewSpacer(), titleLabel, layout.NewSpacer()),
		container.New(layout.NewFormLayout(),
			widget.NewLabel("Nickname"),
			nickField,
			widget.NewLabel("Hostname"),
			hostField,
			widget.NewLabel("Alt. Hostname"),
			altHostField,
			widget.NewLabel("Username"),
			userField,
			widget.NewLabel("Password"),
			a.passField,
		),
		container.NewHBox(layout.NewSpacer(), legacyAuthCheck),
		widget.NewSeparator(),
		container.NewHBox(
			a.promptText,
			layout.NewSpacer(),
			a.submitBtn),
	)
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
