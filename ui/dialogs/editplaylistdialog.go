package dialogs

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
)

type EditPlaylistDialog struct {
	widget.BaseWidget

	OnCanceled       func()
	OnDeletePlaylist func()
	OnUpdateMetadata func()

	IsPublic    bool
	Name        string
	Description string

	container *fyne.Container
}

func NewEditPlaylistDialog(playlist *mediaprovider.Playlist, showPublicCheck bool) *EditPlaylistDialog {
	e := &EditPlaylistDialog{
		IsPublic:    playlist.Public,
		Name:        playlist.Name,
		Description: playlist.Description,
	}
	e.ExtendBaseWidget(e)

	isPublicCheck := widget.NewCheckWithData(lang.L("Public"), binding.BindBool(&e.IsPublic))
	isPublicCheck.Hidden = !showPublicCheck
	nameEntry := widget.NewEntryWithData(binding.BindString(&e.Name))
	descriptionEntry := widget.NewEntryWithData(binding.BindString(&e.Description))
	deleteBtn := widget.NewButtonWithIcon(lang.L("Delete Playlist"), theme.DeleteIcon(), func() {
		if e.OnDeletePlaylist != nil {
			e.OnDeletePlaylist()
		}
	})
	submitBtn := widget.NewButtonWithIcon(lang.L("OK"), theme.ConfirmIcon(), func() {
		if e.OnUpdateMetadata != nil {
			e.OnUpdateMetadata()
		}
	})
	submitBtn.Importance = widget.HighImportance
	cancelBtn := widget.NewButtonWithIcon(lang.L("Cancel"), theme.CancelIcon(), func() {
		if e.OnCanceled != nil {
			e.OnCanceled()
		}
	})

	title := widget.NewLabel(lang.L("Edit Playlist"))
	title.Alignment = fyne.TextAlignCenter
	title.TextStyle.Bold = true
	e.container = container.NewVBox(
		title,
		container.New(layout.NewFormLayout(),
			widget.NewLabel(lang.L("Name")),
			nameEntry,
			widget.NewLabel(lang.L("Description")),
			descriptionEntry,
		),
		container.NewHBox(isPublicCheck, layout.NewSpacer(), deleteBtn),
		widget.NewSeparator(),
		container.NewHBox(
			layout.NewSpacer(),
			cancelBtn, submitBtn),
	)

	return e
}

func (e *EditPlaylistDialog) MinSize() fyne.Size {
	return fyne.NewSize(400, e.BaseWidget.MinSize().Height)
}

func (e *EditPlaylistDialog) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(e.container)
}
