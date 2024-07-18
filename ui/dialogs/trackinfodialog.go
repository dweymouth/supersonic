package dialogs

import (
	"fmt"
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

	addFormRow(c, "Title", t.track.Title)

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

	addFormRow(c, "Duration", util.SecondsToTimeString(float64(t.track.Duration)))
	addFormRow(c, "File path", t.track.FilePath)
	addFormRow(c, "Comment", t.track.Comment)
	addFormRow(c, "Year", strconv.Itoa(t.track.Year))
	addFormRow(c, "Track number", strconv.Itoa(t.track.TrackNumber))
	addFormRow(c, "Disc number", strconv.Itoa(t.track.DiscNumber))

	if t.track.BPM > 0 {
		addFormRow(c, "BPM", strconv.Itoa(t.track.BPM))
	}

	addFormRow(c, "Content type", t.track.ContentType)
	addFormRow(c, "Bit rate", fmt.Sprintf("%d kbps", t.track.BitRate))
	addFormRow(c, "File size", util.BytesToSizeString(t.track.Size))
	addFormRow(c, "Play count", strconv.Itoa(t.track.PlayCount))

	if t.track.ReplayGain.TrackPeak > 0 {
		addFormRow(c, "Track gain", fmt.Sprintf("%0.2f dB", t.track.ReplayGain.TrackGain))
		addFormRow(c, "Track peak", fmt.Sprintf("%0.6f", t.track.ReplayGain.TrackPeak))
	}
	if t.track.ReplayGain.AlbumPeak > 0 {
		addFormRow(c, "Album gain", fmt.Sprintf("%0.2f dB", t.track.ReplayGain.AlbumGain))
		addFormRow(c, "Album peak", fmt.Sprintf("%0.6f", t.track.ReplayGain.AlbumPeak))
	}

	title := widget.NewRichTextWithText("Track Info")
	title.Segments[0].(*widget.TextSegment).Style.TextStyle.Bold = true
	dismissBtn := widget.NewButton("Close", func() {
		if t.OnDismiss != nil {
			t.OnDismiss()
		}
	})

	return widget.NewSimpleRenderer(
		container.NewBorder(
			/*top*/ container.NewHBox(layout.NewSpacer(), title, layout.NewSpacer()),
			/*bottom*/ container.NewVBox(
				widget.NewSeparator(),
				container.NewHBox(layout.NewSpacer(), dismissBtn),
			),
			/*left/right*/ nil, nil,
			/*center*/ container.NewScroll(c),
		),
	)
}

func addFormRow(c *fyne.Container, left, right string) {
	c.Add(newFormText(left, true))
	c.Add(newFormText(right, false))
}

func newFormText(text string, leftCol bool) *widget.RichText {
	alignment := fyne.TextAlignLeading
	if leftCol {
		alignment = fyne.TextAlignTrailing
	}
	rt := widget.NewRichText(
		&widget.TextSegment{
			Text: text,
			Style: widget.RichTextStyle{
				TextStyle: fyne.TextStyle{Bold: leftCol},
				Alignment: alignment,
			},
		},
	)
	if !leftCol {
		rt.Wrapping = fyne.TextWrapWord
	}
	return rt
}
