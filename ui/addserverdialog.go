package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type AddServerDialog struct {
	widget.BaseWidget

	Nickname string
	Host     string
	Username string
	Password string
	OnSubmit func()

	container *fyne.Container
}

var _ fyne.Widget = (*AddServerDialog)(nil)

func NewAddServerDialog(title string) *AddServerDialog {
	a := &AddServerDialog{}
	a.ExtendBaseWidget(a)
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
			layout.NewSpacer(),
			submit))
	return a
}

func (a *AddServerDialog) MinSize() fyne.Size {
	a.ExtendBaseWidget(a)
	return fyne.NewSize(300, a.container.MinSize().Height)
}

func (a *AddServerDialog) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}
