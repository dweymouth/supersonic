package dialogs

import (
	"supersonic/backend"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type AddEditServerDialog struct {
	widget.BaseWidget

	Nickname string
	Host     string
	Username string
	Password string
	OnSubmit func()

	submitBtn  *widget.Button
	promptText *widget.RichText
	container  *fyne.Container
}

var _ fyne.Widget = (*AddEditServerDialog)(nil)

func NewAddEditServerDialog(title string, prefillServer *backend.ServerConfig) *AddEditServerDialog {
	a := &AddEditServerDialog{}
	a.ExtendBaseWidget(a)
	if prefillServer != nil {
		a.Nickname = prefillServer.Nickname
		a.Host = prefillServer.Hostname
		a.Username = prefillServer.Username
	}

	titleLabel := widget.NewLabel(title)
	titleLabel.TextStyle.Bold = true
	nickField := widget.NewEntryWithData(binding.BindString(&a.Nickname))
	nickField.SetPlaceHolder("My Server")
	hostField := widget.NewEntryWithData(binding.BindString(&a.Host))
	hostField.SetPlaceHolder("http://localhost:4533")
	userField := widget.NewEntryWithData(binding.BindString(&a.Username))
	passField := widget.NewPasswordEntry()
	a.submitBtn = widget.NewButton("Enter", func() {
		a.Password = passField.Text
		if a.OnSubmit != nil {
			a.OnSubmit()
		}
	})
	a.promptText = widget.NewRichTextWithText("")
	a.promptText.Hidden = true

	a.container = container.NewVBox(
		container.NewHBox(layout.NewSpacer(), titleLabel, layout.NewSpacer()),
		container.New(layout.NewFormLayout(),
			widget.NewLabel("Nickname"),
			nickField,
			widget.NewLabel("Hostname"),
			hostField,
			widget.NewLabel("Username"),
			userField,
			widget.NewLabel("Password"),
			passField,
		),
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
	return fyne.NewSize(450, a.container.MinSize().Height)
}

func (a *AddEditServerDialog) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}
