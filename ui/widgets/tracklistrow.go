package widgets

import (
	"fmt"
	"image"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"
	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	myTheme "github.com/dweymouth/supersonic/ui/theme"
	"github.com/dweymouth/supersonic/ui/util"
)

var (
	tracklistUpdateCounter = util.NewEventCounter(30)

	emptyTrack = util.TrackListModel{
		Item: &mediaprovider.Track{
			ID:            "dummy",
			Title:         "—",
			ArtistIDs:     []string{"—"},
			ArtistNames:   []string{"—"},
			Album:         "—",
			Genres:        []string{"—"},
			ComposerNames: []string{"—"},
			ComposerIDs:   []string{"—"},
			Comment:       "—",
			FilePath:      "—",
			ContentType:   "—",
		},
	}
)

const tracklistThumbnailSize = 48

type TracklistColumn struct {
	Name string
	Col  ListColumn
}

const (
	ColumnNum         = "Num"
	ColumnTitle       = "Title"
	ColumnArtist      = "Artist"
	ColumnTitleArtist = "Title/Artist"
	ColumnAlbum       = "Album"
	ColumnComposer    = "Composer"
	ColumnTime        = "Time"
	ColumnYear        = "Year"
	ColumnFavorite    = "Favorite"
	ColumnRating      = "Rating"
	ColumnPlays       = "Plays"
	ColumnComment     = "Comment"
	ColumnBPM         = "BPM"
	ColumnBitrate     = "Bitrate"
	ColumnSize        = "Size"
	ColumnPath        = "Path"
)

var (
	ExpandedTracklistRowColumns []TracklistColumn

	ExpandedTracklistRowColumnWidths []float32

	CompactTracklistRowColumns []TracklistColumn

	CompactTracklistRowColumnWidths []float32

	initTracklistColumns = sync.OnceFunc(func() {
		title := lang.L("Title")
		artist := lang.L("Artist")
		album := lang.L("Album")
		composer := lang.L("Composer")
		time := lang.L("Time")
		year := lang.L("Year")
		fav := lang.L("Fav.")
		rating := lang.L("Rating")
		plays := lang.L("Plays")
		comment := lang.L("Comment")
		bpm := lang.L("BPM")
		bitrate := lang.L("Bit rate")
		size := lang.L("Size")
		filepath := lang.L("File path")

		CompactTracklistRowColumns = []TracklistColumn{
			{Name: ColumnNum, Col: ListColumn{Text: "#", Alignment: fyne.TextAlignTrailing, CanToggleVisible: false}},
			{Name: ColumnTitle, Col: ListColumn{Text: title, Alignment: fyne.TextAlignLeading, CanToggleVisible: false}},
			{Name: ColumnArtist, Col: ListColumn{Text: artist, Alignment: fyne.TextAlignLeading, CanToggleVisible: true}},
			{Name: ColumnAlbum, Col: ListColumn{Text: album, Alignment: fyne.TextAlignLeading, CanToggleVisible: true}},
			{Name: ColumnComposer, Col: ListColumn{Text: composer, Alignment: fyne.TextAlignLeading, CanToggleVisible: true}},
			{Name: ColumnTime, Col: ListColumn{Text: time, Alignment: fyne.TextAlignTrailing, CanToggleVisible: true}},
			{Name: ColumnYear, Col: ListColumn{Text: year, Alignment: fyne.TextAlignTrailing, CanToggleVisible: true}},
			{Name: ColumnFavorite, Col: ListColumn{Text: " " + fav, Alignment: fyne.TextAlignCenter, CanToggleVisible: true}},
			{Name: ColumnRating, Col: ListColumn{Text: rating, Alignment: fyne.TextAlignLeading, CanToggleVisible: true}},
			{Name: ColumnPlays, Col: ListColumn{Text: plays, Alignment: fyne.TextAlignTrailing, CanToggleVisible: true}},
			{Name: ColumnComment, Col: ListColumn{Text: comment, Alignment: fyne.TextAlignLeading, CanToggleVisible: true}},
			{Name: ColumnBPM, Col: ListColumn{Text: bpm, Alignment: fyne.TextAlignTrailing, CanToggleVisible: true}},
			{Name: ColumnBitrate, Col: ListColumn{Text: bitrate, Alignment: fyne.TextAlignTrailing, CanToggleVisible: true}},
			{Name: ColumnSize, Col: ListColumn{Text: size, Alignment: fyne.TextAlignTrailing, CanToggleVisible: true}},
			{Name: ColumnPath, Col: ListColumn{Text: filepath, Alignment: fyne.TextAlignLeading, CanToggleVisible: true}},
		}

		ExpandedTracklistRowColumns = []TracklistColumn{
			{Name: ColumnNum, Col: ListColumn{Text: "#", Alignment: fyne.TextAlignTrailing, CanToggleVisible: false}},
			{Name: ColumnTitleArtist, Col: ListColumn{Text: title + " / " + artist, Alignment: fyne.TextAlignLeading, CanToggleVisible: false}},
			{Name: ColumnAlbum, Col: ListColumn{Text: album, Alignment: fyne.TextAlignLeading, CanToggleVisible: true}},
			{Name: ColumnComposer, Col: ListColumn{Text: composer, Alignment: fyne.TextAlignLeading, CanToggleVisible: true}},
			{Name: ColumnTime, Col: ListColumn{Text: time, Alignment: fyne.TextAlignTrailing, CanToggleVisible: true}},
			{Name: ColumnYear, Col: ListColumn{Text: year, Alignment: fyne.TextAlignTrailing, CanToggleVisible: true}},
			{Name: ColumnFavorite, Col: ListColumn{Text: fav, Alignment: fyne.TextAlignCenter, CanToggleVisible: true}},
			{Name: ColumnRating, Col: ListColumn{Text: rating, Alignment: fyne.TextAlignLeading, CanToggleVisible: true}},
			{Name: ColumnPlays, Col: ListColumn{Text: plays, Alignment: fyne.TextAlignTrailing, CanToggleVisible: true}},
			{Name: ColumnComment, Col: ListColumn{Text: comment, Alignment: fyne.TextAlignLeading, CanToggleVisible: true}},
			{Name: ColumnBPM, Col: ListColumn{Text: bpm, Alignment: fyne.TextAlignTrailing, CanToggleVisible: true}},
			{Name: ColumnBitrate, Col: ListColumn{Text: bitrate, Alignment: fyne.TextAlignTrailing, CanToggleVisible: true}},
			{Name: ColumnSize, Col: ListColumn{Text: size, Alignment: fyne.TextAlignTrailing, CanToggleVisible: true}},
			{Name: ColumnPath, Col: ListColumn{Text: filepath, Alignment: fyne.TextAlignLeading, CanToggleVisible: true}},
		}

		var sortIconWidth float32 = 15

		numColWidth := widget.NewLabel("9999").MinSize().Width
		timeColWidth := fyne.Max(
			widget.NewLabel(time).MinSize().Width+sortIconWidth,
			widget.NewLabel("99:99").MinSize().Width)
		yearColWidth := fyne.Max(
			widget.NewLabel(year).MinSize().Width+sortIconWidth,
			widget.NewLabel("2000").MinSize().Width)
		favColWidth := fyne.Max(55,
			widget.NewLabel(fav).MinSize().Width+sortIconWidth)
		ratingColWidth := fyne.Max(100,
			widget.NewLabel(rating).MinSize().Width+sortIconWidth)
		playsColWidth := fyne.Max(
			widget.NewLabel("9999").MinSize().Width,
			widget.NewLabel(plays).MinSize().Width+sortIconWidth)
		bpmColWidth := fyne.Max(
			widget.NewLabel(bpm+"   ").MinSize().Width,
			widget.NewLabel("9999").MinSize().Width)
		bitrateColWidth := fyne.Max(
			widget.NewLabel("1000").MinSize().Width,
			widget.NewLabel(bitrate).MinSize().Width+sortIconWidth)
		sizeColWidth := fyne.Max(
			widget.NewLabel("99.9 MB").MinSize().Width,
			widget.NewLabel(size).MinSize().Width+sortIconWidth)

		// #, Title, Artist, Album, Composer, Time, Year, Favorite, Rating, Plays, Comment, BPM, Bitrate, Size, Path
		CompactTracklistRowColumnWidths = []float32{numColWidth, -1, -1, -1, -1, timeColWidth, yearColWidth, favColWidth, ratingColWidth, playsColWidth, -1, bpmColWidth, bitrateColWidth, sizeColWidth, -1}
		// #, Title/Artist, Album, Composer, Time, Year, Favorite, Rating, Plays, Comment, BPM, Bitrate, Size, Path
		ExpandedTracklistRowColumnWidths = []float32{numColWidth, -1, -1, -1, timeColWidth, yearColWidth, favColWidth, ratingColWidth, playsColWidth, -1, bpmColWidth, bitrateColWidth, sizeColWidth, -1}
	})
)

type tracklistRowBase struct {
	FocusListRowBase

	OnTappedSecondary func(e *fyne.PointEvent, trackIdx int)

	// set by extending widget
	playingIcon fyne.CanvasObject

	// when replacing the content of col 0 with playing icon,
	// the original content is saved here for resetting
	originalNumColContent fyne.CanvasObject

	// internal state
	tracklist  *Tracklist
	trackNum   int
	trackID    string
	coverID    string
	isPlaying  bool
	isFavorite bool
	playCount  int

	nextUpdateModel  *util.TrackListModel
	nextUpdateRowNum int

	num      *widget.Label
	name     *ttwidget.RichText
	artist   *MultiHyperlink
	album    *MultiHyperlink // for disabled support, if albumID is ""
	composer *MultiHyperlink
	dur      *widget.Label
	year     *widget.Label
	favorite *fyne.Container
	rating   *StarRating
	bitrate  *widget.Label
	plays    *widget.Label
	comment  *ttwidget.Label
	bpm      *widget.Label
	size     *widget.Label
	path     *ttwidget.Label

	// must be injected by extending widget
	setColVisibility func(int, bool) bool
}

type TracklistRow interface {
	fyne.CanvasObject
	FocusListRow

	SetOnTappedSecondary(func(_ *fyne.PointEvent, trackNum int))

	TrackID() string
	Update(model *util.TrackListModel, rowNum int, onDone func())
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
	t.img.ScaleMode = canvas.ImageScaleFastest

	t.imageLoader = util.NewThumbnailLoader(im, func(i image.Image) {
		t.img.SetImage(i, false)
	})
	t.imageLoader.OnBeforeLoad = func() {
		t.img.SetImage(nil, false)
	}

	titleArtistImg := container.NewBorder(nil, nil,
		container.New(layout.NewCustomPaddedLayout(2, 2, 2, -4), t.img) /*left*/, nil,
		container.New(layout.NewCustomPaddedVBoxLayout(theme.Padding()-16),
			t.name, t.artist))

	v := makeVerticallyCentered // func alias
	container := container.New(tracklist.colLayout,
		v(t.num), titleArtistImg, v(t.album), v(t.composer), v(t.dur), v(t.year), v(t.favorite), v(t.rating), v(t.plays), v(t.comment), v(t.bpm), v(t.bitrate), v(t.size), v(t.path))
	t.Content = container
	t.setColVisibility = func(colNum int, vis bool) bool {
		c := container.Objects[colNum].(*fyne.Container)
		wasHidden := c.Hidden
		c.Hidden = vis
		return c.Hidden != wasHidden
	}
	return t
}

func (t *ExpandedTracklistRow) Update(tm *util.TrackListModel, rowNum int, _ func()) {
	if t.trackID != tm.Track().ID && t.img.HaveImage() {
		t.img.SetImage(nil, false)
	}
	t.tracklistRowBase.Update(tm, rowNum, func() {
		t.imageLoader.Load(t.coverID)
	})
}

func NewCompactTracklistRow(tracklist *Tracklist, playingIcon fyne.CanvasObject) *CompactTracklistRow {
	t := &CompactTracklistRow{}
	t.ExtendBaseWidget(t)

	t.tracklistRowBase.create(tracklist)
	t.playingIcon = playingIcon

	t.Content = container.New(tracklist.colLayout,
		t.num, t.name, t.artist, t.album, t.composer, t.dur, t.year, t.favorite, t.rating, t.plays, t.comment, t.bpm, t.bitrate, t.size, t.path)

	colHiddenPtrMap := map[int]*bool{
		2:  &t.artist.Hidden,
		3:  &t.album.Hidden,
		4:  &t.composer.Hidden,
		5:  &t.dur.Hidden,
		6:  &t.year.Hidden,
		7:  &t.favorite.Hidden,
		8:  &t.rating.Hidden,
		9:  &t.plays.Hidden,
		10: &t.comment.Hidden,
		11: &t.bpm.Hidden,
		12: &t.bitrate.Hidden,
		13: &t.size.Hidden,
		14: &t.path.Hidden,
	}
	t.setColVisibility = func(colNum int, vis bool) bool {
		ptr, ok := colHiddenPtrMap[colNum]
		if !ok {
			return false // column is always visible
		}
		wasHidden := *ptr
		*ptr = vis
		return vis != wasHidden
	}
	return t
}

func (t *tracklistRowBase) create(tracklist *Tracklist) {
	t.tracklist = tracklist
	t.num = util.NewTrailingAlignLabel()
	t.name = util.NewTruncatingTooltipRichText()
	t.name.OnMouseIn = t.MouseIn
	t.name.OnMouseOut = t.MouseOut
	t.artist = NewMultiHyperlink()
	t.artist.OnTapped = tracklist.onArtistTapped
	t.artist.OnMouseIn = t.MouseIn
	t.artist.OnMouseOut = t.MouseOut
	t.album = NewMultiHyperlink()
	t.album.OnTapped = func(id string) { tracklist.onAlbumTapped(id) }
	t.album.OnMouseIn = t.MouseIn
	t.album.OnMouseOut = t.MouseOut
	t.composer = NewMultiHyperlink()
	t.composer.OnTapped = func(id string) { tracklist.onArtistTapped(id) }
	t.composer.OnMouseIn = t.MouseIn
	t.composer.OnMouseOut = t.MouseOut
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
	t.comment = util.NewTruncatingTooltipLabel()
	t.comment.OnMouseIn = t.MouseIn
	t.comment.OnMouseOut = t.MouseOut
	t.bpm = util.NewTrailingAlignLabel()
	t.bitrate = util.NewTrailingAlignLabel()
	t.size = util.NewTrailingAlignLabel()
	t.path = util.NewTruncatingTooltipLabel()
	t.path.OnMouseIn = t.MouseIn
	t.path.OnMouseOut = t.MouseOut
}

func (t *tracklistRowBase) SetOnTappedSecondary(f func(*fyne.PointEvent, int)) {
	t.OnTappedSecondary = f
}

func (t *tracklistRowBase) TrackID() string {
	return t.trackID
}

func (t *tracklistRowBase) Update(tm *util.TrackListModel, rowNum int, onUpdate func()) {
	if tracklistUpdateCounter.NumEventsSince(time.Now().Add(-150*time.Millisecond)) > 20 {
		t.doUpdate(&emptyTrack, 1)
		if t.nextUpdateModel == nil {
			// queue to run later
			go func() {
				<-time.After(10 * time.Millisecond)
				fyne.Do(func() {
					if t.nextUpdateModel != nil {
						t.doUpdate(t.nextUpdateModel, t.nextUpdateRowNum)
						onUpdate()
					}
					t.nextUpdateModel = nil
				})
			}()
		}
		t.nextUpdateModel = tm
		t.nextUpdateRowNum = rowNum
	} else {
		t.nextUpdateModel = nil
		t.doUpdate(tm, rowNum)
		onUpdate()
	}
}

func (t *tracklistRowBase) doUpdate(tm *util.TrackListModel, rowNum int) {
	changed := false
	if tm.Selected != t.Selected {
		t.Selected = tm.Selected
		changed = true
	}

	// Update info that can change if this row is bound to
	// a new track (*mediaprovider.Track)
	tr := tm.Track()
	if id := tr.ID; id != t.trackID {
		t.EnsureUnfocused()
		t.trackID = id
		t.coverID = tr.CoverArtID

		t.name.Segments[0].(*widget.TextSegment).Text = tr.Title
		t.name.SetToolTip(tr.Title)
		t.artist.BuildSegments(tr.ArtistNames, tr.ArtistIDs)
		t.album.BuildSegments([]string{tr.Album}, []string{tr.AlbumID})
		t.composer.BuildSegments(tr.ComposerNames, tr.ComposerIDs)
		t.dur.Text = util.SecondsToMMSS(tr.Duration.Seconds())
		t.year.Text = strconv.Itoa(tr.Year)
		t.plays.Text = strconv.Itoa(int(tr.PlayCount))
		t.comment.Text = strings.ReplaceAll(tr.Comment, "\n", " ")
		t.comment.SetToolTip(tr.Comment)
		t.bpm.Text = strconv.Itoa(tr.BPM)
		t.bitrate.Text = strconv.Itoa(tr.BitRate)
		t.size.Text = util.BytesToSizeString(tr.Size)
		t.path.Text = tr.FilePath
		t.path.SetToolTip(tr.FilePath)
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
			t.originalNumColContent = t.Content.(*fyne.Container).Objects[0]
			t.Content.(*fyne.Container).Objects[0] = t.playingIcon
		} else {
			t.Content.(*fyne.Container).Objects[0] = t.originalNumColContent
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
	for i := 2; i < len(t.tracklist.columns); i++ {
		if ch := t.setColVisibility(i, !t.tracklist.visibleColumns[i]); ch {
			changed = true
		}
	}

	if changed {
		tracklistUpdateCounter.Add()
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
