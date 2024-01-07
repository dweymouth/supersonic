package util

import (
	"fmt"
	"image"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/res"
	"github.com/dweymouth/supersonic/sharedutil"
	"github.com/dweymouth/supersonic/ui/layouts"
	myTheme "github.com/dweymouth/supersonic/ui/theme"
	"golang.org/x/net/html"
)

var BoldRichTextStyle = widget.RichTextStyle{TextStyle: fyne.TextStyle{Bold: true}, Inline: true}

func SecondsToTimeString(s float64) string {
	if s < 0 {
		s = 0
	}
	sec := int(math.Round(s))
	min := sec / 60
	sec -= min * 60

	return fmt.Sprintf("%d:%02d", min, sec)
}

func BytesToSizeString(bytes int64) string {
	var num float64
	var suffix string
	switch b := float64(bytes); {
	case b > 1000000000:
		suffix = "GB"
		num = b / 1000000000
	case b > 1000000:
		suffix = "MB"
		num = b / 1000000
	case b > 1000:
		suffix = "KB"
		num = b / 1000
	default:
		suffix = "B"
		num = b
	}
	return fmt.Sprintf(fmtStringForThreeSigFigs(num)+" %s", num, suffix)
}

func fmtStringForThreeSigFigs(num float64) string {
	switch {
	case num >= 100:
		return "%0.0f"
	case num >= 10:
		return "%0.1f"
	default:
		return "%0.2f"
	}
}

func ImageAspect(im image.Image) float32 {
	b := im.Bounds()
	return float32(b.Max.X-b.Min.X) / float32(b.Max.Y-b.Min.Y)
}

// Debouncer returns a function that will call callOnDone when
// it has not been invoked since the last dur interval.
func NewDebouncer(dur time.Duration, callOnDone func()) func() {
	var mu sync.Mutex
	var timer *time.Timer
	return func() {
		mu.Lock()
		defer mu.Unlock()

		if timer != nil {
			timer.Stop()
		}
		timer = time.AfterFunc(dur, callOnDone)
	}
}

func PlaintextFromHTMLString(s string) string {
	tokr := html.NewTokenizer(strings.NewReader(s))

	var text string
	var isLink bool
	var done bool
	for !done {
		tt := tokr.Next()
		switch {
		case tt == html.ErrorToken:
			done = true
		case tt == html.StartTagToken:
			t := tokr.Token()
			isLink = t.Data == "a"
		case tt == html.EndTagToken:
			isLink = false
		case tt == html.TextToken:
			t := tokr.Token()
			// for now, skip displaying Navidrome's "Read more on Last.FM" link
			if !isLink {
				text = text + t.Data
			}
		}
	}
	return text
}

func DisplayReleaseType(releaseTypes mediaprovider.ReleaseTypes) string {
	baseType := "Album"
	switch {
	case releaseTypes&mediaprovider.ReleaseTypeAudiobook > 0:
		baseType = "Audiobook"
	case releaseTypes&mediaprovider.ReleaseTypeAudioDrama > 0:
		baseType = "Audio Drama"
	case releaseTypes&mediaprovider.ReleaseTypeBroadcast > 0:
		baseType = "Broadcast"
	case releaseTypes&mediaprovider.ReleaseTypeDJMix > 0:
		baseType = "DJ-Mix"
	case releaseTypes&mediaprovider.ReleaseTypeEP > 0:
		baseType = "EP"
	case releaseTypes&mediaprovider.ReleaseTypeFieldRecording > 0:
		baseType = "Field Recording"
	case releaseTypes&mediaprovider.ReleaseTypeInterview > 0:
		baseType = "Interview"
	case releaseTypes&mediaprovider.ReleaseTypeMixtape > 0:
		baseType = "Mixtape"
	case releaseTypes&mediaprovider.ReleaseTypeSingle > 0:
		baseType = "Single"
	case releaseTypes&mediaprovider.ReleaseTypeSoundtrack > 0:
		baseType = "Soundtrack"
	}

	var modifiers []string
	if releaseTypes&mediaprovider.ReleaseTypeLive > 0 {
		modifiers = append(modifiers, "Live")
	}
	if releaseTypes&mediaprovider.ReleaseTypeDemo > 0 {
		modifiers = append(modifiers, "Demo")
	}
	if releaseTypes&mediaprovider.ReleaseTypeRemix > 0 {
		modifiers = append(modifiers, "Remix")
	}
	if releaseTypes&mediaprovider.ReleaseTypeSpokenWord > 0 {
		modifiers = append(modifiers, "Spoken Word")
	}
	if releaseTypes&mediaprovider.ReleaseTypeCompilation > 0 {
		modifiers = append(modifiers, "Compilation")
	}

	modifiers = append(modifiers, baseType)
	return strings.Join(modifiers, " ")
}

func NewRatingSubmenu(onSetRating func(int)) *fyne.MenuItem {
	newRatingMenuItem := func(rating int) *fyne.MenuItem {
		label := "(none)"
		if rating > 0 {
			label = strconv.Itoa(rating)
		}
		return fyne.NewMenuItem(label, func() {
			onSetRating(rating)
		})
	}
	ratingMenu := fyne.NewMenuItem("Set rating", nil)
	ratingMenu.Icon = theme.NewThemedResource(res.ResStarOutlineSvg)
	ratingMenu.ChildMenu = fyne.NewMenu("", []*fyne.MenuItem{
		newRatingMenuItem(0),
		newRatingMenuItem(1),
		newRatingMenuItem(2),
		newRatingMenuItem(3),
		newRatingMenuItem(4),
		newRatingMenuItem(5),
	}...)
	return ratingMenu
}

func NewReorderTracksSubmenu(onReorderTracks func(sharedutil.TrackReorderOp)) *fyne.MenuItem {
	reorderMenu := fyne.NewMenuItem("Reorder tracks", nil)
	reorderMenu.Icon = myTheme.SortIcon
	reorderMenu.ChildMenu = fyne.NewMenu("", []*fyne.MenuItem{
		fyne.NewMenuItem("Move to top", func() { onReorderTracks(sharedutil.MoveToTop) }),
		fyne.NewMenuItem("Move up", func() { onReorderTracks(sharedutil.MoveUp) }),
		fyne.NewMenuItem("Move down", func() { onReorderTracks(sharedutil.MoveDown) }),
		fyne.NewMenuItem("Move to bottom", func() { onReorderTracks(sharedutil.MoveToBottom) }),
	}...)
	return reorderMenu
}

func AddHeaderBackground(obj fyne.CanvasObject) *fyne.Container {
	bgrnd := myTheme.NewThemedRectangle(myTheme.ColorNamePageHeader)
	bgrnd.CornerRadiusName = theme.SizeNameInputRadius
	return container.NewStack(bgrnd,
		container.New(&layouts.MaxPadLayout{PadLeft: 10, PadRight: 10, PadTop: 10, PadBottom: 10},
			obj))
}

type HSpace struct {
	widget.BaseWidget

	Width float32
}

func NewHSpace(w float32) *HSpace {
	h := &HSpace{Width: w}
	h.ExtendBaseWidget(h)
	return h
}

func (h *HSpace) MinSize() fyne.Size {
	return fyne.NewSize(h.Width, 0)
}

func (h *HSpace) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(layout.NewSpacer())
}
