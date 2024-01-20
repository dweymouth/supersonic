package dialogs

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type AddToPlaylistDialog struct {
	widget.BaseWidget

	OnCanceled func()
	OnSubmit   func(playlistChoice int, newPlaylistName string)

	playlistSelect   *widget.Select
	newPlaylistLabel *widget.Label
	newPlaylistName  *widget.Entry
	okBtn            *widget.Button

	container *fyne.Container
}

var _ fyne.Widget = (*AddToPlaylistDialog)(nil)

func NewAddToPlaylistDialog(title string, existingPlaylistNames []string, selectedIdx int) *AddToPlaylistDialog {
	a := &AddToPlaylistDialog{}
	a.ExtendBaseWidget(a)

	titleLabel := widget.NewLabel(title)
	titleLabel.TextStyle.Bold = true
	options := []string{"New playlist..."}
	options = append(options, existingPlaylistNames...)
	a.playlistSelect = widget.NewSelect(options, func(_ string) {
		a.onSelectionChanged()
	})
	a.playlistSelect.PlaceHolder = "(Choose playlist)"
	if selectedIdx >= 0 {
		// calling SetSelectedIndex before showing the Select crashes...
		go func() {
			time.Sleep(10 * time.Millisecond)
			// add 1 to selectedIdx to account for "(Choose playlist)" entry
			a.playlistSelect.SetSelectedIndex(selectedIdx + 1)
		}()
	}
	a.newPlaylistName = widget.NewEntry()
	a.newPlaylistName.Hidden = true
	a.newPlaylistName.OnChanged = func(text string) {
		if len(text) > 0 {
			a.okBtn.Enable()
		} else {
			a.okBtn.Disable()
		}
	}
	a.newPlaylistLabel = widget.NewLabel("Name")
	a.newPlaylistLabel.Hidden = true

	a.okBtn = widget.NewButton("OK", a.onOK)
	a.okBtn.Importance = widget.HighImportance
	a.okBtn.Disable()
	cancelBtn := widget.NewButton("Cancel", a.onCancel)

	a.container = container.NewVBox(
		container.NewHBox(layout.NewSpacer(), titleLabel, layout.NewSpacer()),
		container.New(layout.NewFormLayout(),
			widget.NewLabel("Playlist"),
			a.playlistSelect,
			a.newPlaylistLabel,
			a.newPlaylistName),
		widget.NewSeparator(),
		container.NewHBox(layout.NewSpacer(), a.okBtn, cancelBtn))

	return a
}

func (a *AddToPlaylistDialog) onOK() {
	var newPlaylistName string
	playlistChoice := -1
	if sel := a.playlistSelect.SelectedIndex(); sel == 0 {
		newPlaylistName = a.newPlaylistName.Text
	} else {
		playlistChoice = sel - 1
	}
	if a.OnSubmit != nil {
		a.OnSubmit(playlistChoice, newPlaylistName)
	}
}

func (a *AddToPlaylistDialog) onSelectionChanged() {
	if a.playlistSelect.SelectedIndex() == 0 {
		a.newPlaylistName.Show()
		a.newPlaylistLabel.Show()
		if len(a.newPlaylistName.Text) == 0 {
			a.okBtn.Disable()
		} else {
			a.okBtn.Enable()
		}
	} else {
		a.newPlaylistName.Hide()
		a.newPlaylistLabel.Hide()
		a.okBtn.Enable()
	}
}

func (a *AddToPlaylistDialog) onCancel() {
	if a.OnCanceled != nil {
		a.OnCanceled()
	}
}

func (a *AddToPlaylistDialog) MinSize() fyne.Size {
	a.ExtendBaseWidget(a)
	return fyne.NewSize(300, a.container.MinSize().Height)
}

func (a *AddToPlaylistDialog) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}
