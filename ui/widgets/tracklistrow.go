package widgets

import (
	"fmt"
	"image"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/supersonic/backend"
	myTheme "github.com/dweymouth/supersonic/ui/theme"
	"github.com/dweymouth/supersonic/ui/util"
)

const tracklistThumbnailSize = 48

type tracklistRowBase struct {
	FocusListRowBase

	OnTappedSecondary func(e *fyne.PointEvent, trackIdx int)

	// set by extending widget
	playingIcon fyne.CanvasObject

	// internal state
	tracklist  *Tracklist
	trackNum   int
	trackID    string
	isPlaying  bool
	isFavorite bool
	playCount  int

	num      *widget.Label
	name     *widget.RichText // for bold support
	artist   *MultiHyperlink
	album    *MultiHyperlink // for disabled support, if albumID is ""
	dur      *widget.Label
	year     *widget.Label
	favorite *fyne.Container
	rating   *StarRating
	bitrate  *widget.Label
	plays    *widget.Label
	comment  *widget.Label
	size     *widget.Label
	path     *widget.Label
}

type TracklistRow interface {
	fyne.CanvasObject
	FocusListRow

	SetOnTappedSecondary(func(_ *fyne.PointEvent, trackNum int))

	TrackID() string
	Update(model *util.TrackListModel, rowNum int)
}

type ExpandedTracklistRow struct {
	tracklistRowBase

	img         *ImagePlaceholder
	imageLoader util.ThumbnailLoader
}

type CompactTracklistRow struct {
	tracklistRowBase
}

var (
	_ TracklistRow = (*CompactTracklistRow)(nil)
	_ TracklistRow = (*ExpandedTracklistRow)(nil)
)

func NewExpandedTracklistRow(tracklist *Tracklist, im *backend.ImageManager, playingIcon fyne.CanvasObject) *ExpandedTracklistRow {
	t := &ExpandedTracklistRow{}
	t.ExtendBaseWidget(t)
	t.tracklistRowBase.create(tracklist)
	t.playingIcon = playingIcon
	t.img = NewImagePlaceholder(myTheme.TracksIcon, tracklistThumbnailSize)

	t.imageLoader = util.NewThumbnailLoader(im, func(i image.Image) {
		t.img.SetImage(i, false)
	})
	t.imageLoader.OnBeforeLoad = func() {
		t.img.SetImage(nil, false)
	}

	titleArtistImg := container.NewBorder(nil, nil,
		container.NewPadded(t.img) /*left*/, nil,
		container.New(layout.NewCustomPaddedVBoxLayout(theme.Padding()-16),
			t.name, t.artist))

	v := makeVerticallyCentered // func alias
	t.Content = container.New(tracklist.colLayout,
		v(t.num), titleArtistImg, v(t.album), v(t.dur), v(t.year), v(t.favorite), v(t.rating), v(t.plays), v(t.comment), v(t.bitrate), v(t.size), v(t.path))
	return t
}

func (t *ExpandedTracklistRow) Update(tm *util.TrackListModel, rowNum int) {
	if t.trackID != tm.Track.ID {
		t.imageLoader.Load(tm.Track.CoverArtID)
	}
	t.tracklistRowBase.Update(tm, rowNum)
}

func NewCompactTracklistRow(tracklist *Tracklist, playingIcon fyne.CanvasObject) *CompactTracklistRow {
	t := &CompactTracklistRow{}
	t.ExtendBaseWidget(t)
	t.tracklistRowBase.create(tracklist)
	t.playingIcon = playingIcon

	t.Content = container.New(tracklist.colLayout,
		t.num, t.name, t.artist, t.album, t.dur, t.year, t.favorite, t.rating, t.plays, t.comment, t.bitrate, t.size, t.path)
	return t
}

func (t *tracklistRowBase) create(tracklist *Tracklist) {
	t.tracklist = tracklist
	t.num = util.NewTrailingAlignLabel()
	t.name = util.NewTruncatingRichText()
	t.artist = NewMultiHyperlink()
	t.artist.OnTapped = tracklist.onArtistTapped
	t.album = NewMultiHyperlink()
	t.album.OnTapped = func(id string) { tracklist.onAlbumTapped(id) }
	t.dur = util.NewTrailingAlignLabel()
	t.year = util.NewTrailingAlignLabel()
	favorite := NewFavoriteIcon()
	favorite.OnTapped = t.toggleFavorited
	t.favorite = container.NewCenter(favorite)
	t.rating = NewStarRating()
	t.rating.IsDisabled = t.tracklist.Options.DisableRating
	t.rating.StarSize = 16
	t.rating.OnRatingChanged = t.setTrackRating
	t.plays = util.NewTrailingAlignLabel()
	t.comment = util.NewTruncatingLabel()
	t.bitrate = util.NewTrailingAlignLabel()
	t.size = util.NewTrailingAlignLabel()
	t.path = util.NewTruncatingLabel()
}

func (t *tracklistRowBase) SetOnTappedSecondary(f func(*fyne.PointEvent, int)) {
	t.OnTappedSecondary = f
}
func (t *tracklistRowBase) TrackID() string {
	return t.trackID
}

func (t *tracklistRowBase) Update(tm *util.TrackListModel, rowNum int) {
	changed := false
	if tm.Selected != t.Selected {
		t.Selected = tm.Selected
		changed = true
	}

	// Update info that can change if this row is bound to
	// a new track (*mediaprovider.Track)
	tr := tm.Track
	if tr.ID != t.trackID {
		t.EnsureUnfocused()
		t.trackID = tr.ID

		t.name.Segments[0].(*widget.TextSegment).Text = tr.Name
		t.artist.BuildSegments(tr.ArtistNames, tr.ArtistIDs)
		t.album.BuildSegments([]string{tr.Album}, []string{tr.AlbumID})
		t.dur.Text = util.SecondsToTimeString(float64(tr.Duration))
		t.year.Text = strconv.Itoa(tr.Year)
		t.plays.Text = strconv.Itoa(int(tr.PlayCount))
		t.comment.Text = tr.Comment
		t.bitrate.Text = strconv.Itoa(tr.BitRate)
		t.size.Text = util.BytesToSizeString(tr.Size)
		t.path.Text = tr.FilePath
		changed = true
	}

	// Update track num if needed
	// (which can change based on bound *mediaprovider.Track or tracklist.AutoNumber)
	if t.trackNum != rowNum {
		discNum := -1
		var str string
		if rowNum < 0 {
			rowNum = tr.TrackNumber
			if t.tracklist.Options.ShowDiscNumber {
				discNum = tr.DiscNumber
			}
		}
		t.trackNum = rowNum
		if discNum >= 0 {
			str = fmt.Sprintf("%d.%02d", discNum, rowNum)
		} else {
			str = strconv.Itoa(rowNum)
		}
		t.num.Text = str
		changed = true
	}

	// Update play count if needed
	if tr.PlayCount != t.playCount {
		t.playCount = tr.PlayCount
		t.plays.Text = strconv.Itoa(int(tr.PlayCount))
		changed = true
	}

	// Render whether track is playing or not
	if isPlaying := t.tracklist.nowPlayingID == tr.ID; isPlaying != t.isPlaying {
		t.isPlaying = isPlaying
		t.name.Segments[0].(*widget.TextSegment).Style.TextStyle.Bold = isPlaying

		if isPlaying {
			t.Content.(*fyne.Container).Objects[0] = t.playingIcon
		} else {
			t.Content.(*fyne.Container).Objects[0] = t.num
		}
		changed = true
	}

	// Update favorite column
	if tr.Favorite != t.isFavorite {
		t.isFavorite = tr.Favorite
		t.favorite.Objects[0].(*FavoriteIcon).Favorite = tr.Favorite
		changed = true
	}

	// Update rating column
	if t.rating.Rating != tr.Rating {
		t.rating.Rating = tr.Rating
		t.rating.Refresh()
	}
	if t.rating.IsDisabled != t.tracklist.Options.DisableRating {
		t.rating.IsDisabled = t.tracklist.Options.DisableRating
		t.rating.Refresh()
	}

	// Show only columns configured to be visible
	updateHidden := func(hiddenPtr *bool, colName string) {
		colHidden := !t.tracklist.visibleColumns[ColNumber(colName)]
		if colHidden != *hiddenPtr {
			*hiddenPtr = colHidden
			changed = true
		}
	}
	updateHidden(&t.artist.Hidden, ColumnArtist)
	updateHidden(&t.album.Hidden, ColumnAlbum)
	updateHidden(&t.dur.Hidden, ColumnTime)
	updateHidden(&t.year.Hidden, ColumnYear)
	updateHidden(&t.favorite.Hidden, ColumnFavorite)
	updateHidden(&t.rating.Hidden, ColumnRating)
	updateHidden(&t.plays.Hidden, ColumnPlays)
	updateHidden(&t.comment.Hidden, ColumnComment)
	updateHidden(&t.bitrate.Hidden, ColumnBitrate)
	updateHidden(&t.size.Hidden, ColumnSize)
	updateHidden(&t.path.Hidden, ColumnPath)

	if changed {
		t.Refresh()
	}
}

func (t *tracklistRowBase) toggleFavorited() {
	t.isFavorite = !t.isFavorite
	favIcon := t.favorite.Objects[0].(*FavoriteIcon)
	favIcon.Favorite = t.isFavorite
	t.favorite.Refresh()
	t.tracklist.onSetFavorite(t.trackID, t.isFavorite)
}

func (t *tracklistRowBase) setTrackRating(rating int) {
	t.tracklist.onSetRating(t.trackID, rating)
}

func (t *tracklistRowBase) TappedSecondary(e *fyne.PointEvent) {
	if t.OnTappedSecondary != nil {
		t.OnTappedSecondary(e, t.ListItemID)
	}
}

func makeVerticallyCentered(obj fyne.CanvasObject) fyne.CanvasObject {
	return container.NewVBox(layout.NewSpacer(), obj, layout.NewSpacer())
}
