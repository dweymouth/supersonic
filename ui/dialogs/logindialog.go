package dialogs

import (
	"github.com/dweymouth/supersonic/backend"

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

	servers      []*backend.ServerConfig
	serverSelect *widget.Select
	passField    *widget.Entry
	promptText   *widget.RichText
	submitBtn    *widget.Button

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
	l.submitBtn = widget.NewButton("OK", l.onSubmit)

	l.promptText = widget.NewRichTextWithText("")
	l.promptText.Segments[0].(*widget.TextSegment).Style.ColorName = theme.ColorNameError
	l.promptText.Hidden = true

	l.container = container.NewVBox(
		container.NewHBox(layout.NewSpacer(), titleLabel, layout.NewSpacer()),
		container.New(layout.NewFormLayout(),
			widget.NewLabel("Server"),
			container.NewBorder(nil, nil, nil, editBtn, l.serverSelect),
			widget.NewLabel("Password"),
			l.passField),
		widget.NewSeparator(),
		container.NewHBox(l.promptText, layout.NewSpacer(), l.submitBtn),
	)
	return l
}

func (l *LoginDialog) SetInfoText(text string) {
	l.doSetPromptText(text, theme.ColorNameForeground)
}

func (l *LoginDialog) SetErrorText(text string) {
	l.doSetPromptText(text, theme.ColorNameError)
}

func (l *LoginDialog) EnableSubmit() {
	l.submitBtn.Enable()
	l.submitBtn.Refresh()
}

func (l *LoginDialog) DisableSubmit() {
	l.submitBtn.Disable()
	l.submitBtn.Refresh()
}

func (l *LoginDialog) doSetPromptText(text string, color fyne.ThemeColorName) {
	ts := l.promptText.Segments[0].(*widget.TextSegment)
	ts.Text = text
	ts.Style.ColorName = color
	if text != "" {
		l.promptText.Show()
	} else {
		l.promptText.Hide()
	}
	l.promptText.Refresh()
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
