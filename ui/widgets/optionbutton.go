package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type OptionButton struct {
	widget.BaseWidget

	Text     string
	Icon     fyne.Resource
	Menu     *fyne.Menu
	OnTapped func()
}

func NewOptionButton(text string, menu *fyne.Menu, onTapped func()) *OptionButton {
	return NewOptionButtonWithIcon(text, nil, menu, onTapped)
}

func NewOptionButtonWithIcon(text string, icon fyne.Resource, menu *fyne.Menu, onTapped func()) *OptionButton {
	o := &OptionButton{
		Menu:     menu,
		Text:     text,
		Icon:     icon,
		OnTapped: onTapped,
	}
	o.ExtendBaseWidget(o)
	return o
}

func (o *OptionButton) CreateRenderer() fyne.WidgetRenderer {
	return newOptionButtonRenderer(o)
}

var _ fyne.WidgetRenderer = (*optionButtonRenderer)(nil)

type optionButtonRenderer struct {
	wid         *OptionButton
	mainBtn     *widget.Button
	auxBtn      *widget.Button
	dividerLine *canvas.Rectangle
	objects     []fyne.CanvasObject
}

func newOptionButtonRenderer(o *OptionButton) *optionButtonRenderer {
	render := &optionButtonRenderer{wid: o}
	render.mainBtn = widget.NewButtonWithIcon(o.Text, o.Icon, render.onMainBtnTapped)
	render.mainBtn.Alignment = widget.ButtonAlignLeading
	render.auxBtn = widget.NewButtonWithIcon("", theme.MenuDropDownIcon(), render.showMenu)
	render.auxBtn.Importance = widget.LowImportance

	render.dividerLine = canvas.NewRectangle(o.Theme().Color(theme.ColorNameSeparator,
		fyne.CurrentApp().Settings().ThemeVariant()))
	render.dividerLine.SetMinSize(fyne.NewSquareSize(1))
	divider := container.NewBorder(layout.NewSpacer(), layout.NewSpacer(), nil, nil, render.dividerLine)

	render.objects = []fyne.CanvasObject{
		container.NewStack(
			render.mainBtn,
			container.New(layout.NewCustomPaddedHBoxLayout(0),
				layout.NewSpacer(), divider, render.auxBtn),
		),
	}

	return render
}

func (o *optionButtonRenderer) showMenu() {
	if o.wid.Menu == nil {
		return
	}

	canv := fyne.CurrentApp().Driver().CanvasForObject(o.wid)
	pop := widget.NewPopUpMenu(o.wid.Menu, canv)
	pop.ShowAtRelativePosition(o.auxBtn.Position().Add(
		fyne.NewPos(0, o.wid.Size().Height)),
		o.wid,
	)
}

func (o *optionButtonRenderer) MinSize() fyne.Size {
	return o.mainBtn.MinSize().Add(fyne.NewSize(o.auxBtn.MinSize().Width, 0))
}

func (o *optionButtonRenderer) Layout(s fyne.Size) {
	o.objects[0].(*fyne.Container).Resize(s)
}

func (o *optionButtonRenderer) Objects() []fyne.CanvasObject {
	return o.objects
}

func (o *optionButtonRenderer) Refresh() {
	o.mainBtn.Text = o.wid.Text
	o.mainBtn.Icon = o.wid.Icon
	o.dividerLine.FillColor = o.wid.Theme().Color(theme.ColorNameSeparator, fyne.CurrentApp().Settings().ThemeVariant())
	o.objects[0].Refresh()
}

func (o *optionButtonRenderer) Destroy() {}

func (o *optionButtonRenderer) onMainBtnTapped() {
	if o.wid.OnTapped != nil {
		o.wid.OnTapped()
	}
}
