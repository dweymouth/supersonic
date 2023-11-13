package dialogs

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
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

	isPublicCheck := widget.NewCheckWithData("Public", binding.BindBool(&e.IsPublic))
	isPublicCheck.Hidden = !showPublicCheck
	nameEntry := widget.NewEntryWithData(binding.BindString(&e.Name))
	descriptionEntry := widget.NewEntryWithData(binding.BindString(&e.Description))
	deleteBtn := widget.NewButton("Delete Playlist", func() {
		if e.OnDeletePlaylist != nil {
			e.OnDeletePlaylist()
		}
	})
	submitBtn := widget.NewButton("OK", func() {
		if e.OnUpdateMetadata != nil {
			e.OnUpdateMetadata()
		}
	})
	submitBtn.Importance = widget.HighImportance
	cancelBtn := widget.NewButton("Cancel", func() {
		if e.OnCanceled != nil {
			e.OnCanceled()
		}
	})

	e.container = container.NewVBox(
		container.NewHBox(layout.NewSpacer(), widget.NewLabel("Edit Playlist"), layout.NewSpacer()),
		container.New(layout.NewFormLayout(),
			widget.NewLabel("Name"),
			nameEntry,
			widget.NewLabel("Description"),
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
	return fyne.NewSize(300, e.BaseWidget.MinSize().Height)
}

func (e *EditPlaylistDialog) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(e.container)
}
