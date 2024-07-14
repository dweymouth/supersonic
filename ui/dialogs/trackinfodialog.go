package dialogs

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/ui/util"
	"github.com/dweymouth/supersonic/ui/widgets"
)

type TrackInfoDialog struct {
	widget.BaseWidget

	OnDismiss          func()
	OnNavigateToArtist func(artistID string)
	OnNavigateToAlbum  func(albumID string)

	track *mediaprovider.Track
}

func NewTrackInfoDialog(track *mediaprovider.Track) *TrackInfoDialog {
	t := &TrackInfoDialog{track: track}
	t.ExtendBaseWidget(t)
	return t
}

func (t *TrackInfoDialog) CreateRenderer() fyne.WidgetRenderer {
	c := container.New(layout.NewFormLayout())
	c.Add(newFormText("Title", true))
	c.Add(newFormText(t.track.Title, false))

	c.Add(newFormText("Artist", true))
	artists := widgets.NewMultiHyperlink()
	artists.BuildSegments(t.track.ArtistNames, t.track.ArtistIDs)
	artists.OnTapped = func(id string) {
		if t.OnNavigateToArtist != nil {
			t.OnNavigateToArtist(id)
		}
	}
	c.Add(artists)

	c.Add(newFormText("Album", true))
	album := widget.NewHyperlink(t.track.Album, nil)
	album.OnTapped = func() {
		if t.OnNavigateToAlbum != nil {
			t.OnNavigateToAlbum(t.track.AlbumID)
		}
	}
	c.Add(album)

	c.Add(newFormText("File Path", true))
	path := widgets.NewMaxRowsLabel(2, t.track.FilePath)
	//path.Segments[0].(*widget.TextSegment).Style.SizeName = myTheme.SizeNameSubText
	path.Wrapping = fyne.TextWrapWord
	path.Truncation = fyne.TextTruncateEllipsis
	c.Add(path)

	c.Add(newFormText("Comment", true))
	c.Add(newFormText(t.track.Comment, false))

	c.Add(newFormText("Year", true))
	c.Add(newFormText(strconv.Itoa(t.track.Year), false))

	c.Add(newFormText("Bit Rate", true))
	c.Add(newFormText(strconv.Itoa(t.track.BitRate)+"kbps", false))

	c.Add(newFormText("File Size", true))
	c.Add(newFormText(util.BytesToSizeString(t.track.Size), false))

	c.Add(newFormText("Play Count", true))
	c.Add(newFormText(strconv.Itoa(t.track.PlayCount), false))

	return widget.NewSimpleRenderer(c)
}

func newFormText(text string, leftCol bool) *widget.RichText {
	alignment := fyne.TextAlignLeading
	if leftCol {
		alignment = fyne.TextAlignTrailing
	}
	return widget.NewRichText(
		&widget.TextSegment{
			Text: text,
			Style: widget.RichTextStyle{
				TextStyle: fyne.TextStyle{Bold: leftCol},
				Alignment: alignment,
			},
		},
	)
}
