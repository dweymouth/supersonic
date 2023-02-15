package dialogs

import (
	"supersonic/backend"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type LoginDialog struct {
	widget.BaseWidget

	OnSubmit     func(server *backend.ServerConfig, password string)
	OnEditServer func(server *backend.ServerConfig)

	servers       []*backend.ServerConfig
	serverSelect  *widget.Select
	passField     *widget.Entry
	errPromptText *widget.RichText

	container *fyne.Container
}

var _ fyne.Widget = (*LoginDialog)(nil)

func NewLoginDialog(servers []*backend.ServerConfig) *LoginDialog {
	l := &LoginDialog{servers: servers}
	l.ExtendBaseWidget(l)
	titleLabel := widget.NewLabel("Login to Server")
	titleLabel.TextStyle.Bold = true
	serverNames := make([]string, len(servers))
	for i, s := range servers {
		serverNames[i] = s.Nickname
	}
	l.serverSelect = widget.NewSelect(serverNames, func(_ string) {})
	l.serverSelect.SetSelectedIndex(0)
	editBtn := widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), l.onEditServer)
	l.passField = widget.NewPasswordEntry()
	okBtn := widget.NewButton("OK", l.onSubmit)

	l.errPromptText = widget.NewRichTextWithText("")
	l.errPromptText.Segments[0].(*widget.TextSegment).Style.ColorName = theme.ColorNameError
	l.errPromptText.Hidden = true

	l.container = container.NewVBox(
		container.NewHBox(layout.NewSpacer(), titleLabel, layout.NewSpacer()),
		container.New(layout.NewFormLayout(),
			widget.NewLabel("Server"),
			container.NewBorder(nil, nil, nil, editBtn, l.serverSelect),
			widget.NewLabel("Password"),
			l.passField),
		widget.NewSeparator(),
		container.NewHBox(l.errPromptText, layout.NewSpacer(), okBtn),
	)
	return l
}

func (l *LoginDialog) SetErrorText(text string) {
	l.errPromptText.Segments[0].(*widget.TextSegment).Text = text
	if text != "" {
		l.errPromptText.Show()
	} else {
		l.errPromptText.Hide()
	}
	l.errPromptText.Refresh()
}

func (l *LoginDialog) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(l.container)
}

func (l *LoginDialog) MinSize() fyne.Size {
	l.ExtendBaseWidget(l)
	return fyne.NewSize(300, l.container.MinSize().Height)
}

func (l *LoginDialog) onSubmit() {
	if l.OnSubmit != nil {
		l.OnSubmit(l.servers[l.serverSelect.SelectedIndex()], l.passField.Text)
	}
}

func (l *LoginDialog) onEditServer() {
	if l.OnEditServer != nil {
		l.OnEditServer(l.servers[l.serverSelect.SelectedIndex()])
	}
}
