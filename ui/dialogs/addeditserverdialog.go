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

	errPromptText *widget.RichText
	container     *fyne.Container
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
	submit := widget.NewButton("Enter", func() {
		a.Password = passField.Text
		if a.OnSubmit != nil {
			a.OnSubmit()
		}
	})
	a.errPromptText = widget.NewRichTextWithText("")
	a.errPromptText.Segments[0].(*widget.TextSegment).Style.ColorName = theme.ColorNameError
	a.errPromptText.Hidden = true

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
			a.errPromptText,
			layout.NewSpacer(),
			submit),
	)
	return a
}

func (a *AddEditServerDialog) SetErrorText(text string) {
	a.errPromptText.Segments[0].(*widget.TextSegment).Text = text
	if text != "" {
		a.errPromptText.Show()
	} else {
		a.errPromptText.Hide()
	}
	a.errPromptText.Refresh()
}

func (a *AddEditServerDialog) MinSize() fyne.Size {
	a.ExtendBaseWidget(a)
	return fyne.NewSize(450, a.container.MinSize().Height)
}

func (a *AddEditServerDialog) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}
