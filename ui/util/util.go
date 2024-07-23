package util

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/res"
	myTheme "github.com/dweymouth/supersonic/ui/theme"
	"golang.org/x/net/html"
)

var BoldRichTextStyle = widget.RichTextStyle{TextStyle: fyne.TextStyle{Bold: true}, Inline: true}

func MakeOpaque(c color.Color) color.Color {
	if nrgba, ok := c.(color.NRGBA); ok {
		nrgba.A = 255
		return nrgba
	}
	return c
}

func SecondsToMMSS(s float64) string {
	if s < 0 {
		s = 0
	}
	sec := int(math.Round(s))
	min := sec / 60
	sec -= min * 60

	return fmt.Sprintf("%d:%02d", min, sec)
}

func SecondsToTimeString(s float64) string {
	if s < 3600 /*1 hour*/ {
		return SecondsToMMSS(s)
	}
	sec := int64(s)
	days := sec / 86400
	sec -= days * 86400
	hr := sec / 3600
	sec -= hr * 3600
	min := sec / 60
	sec -= min * 60

	var str string
	if days > 0 {
		daysStr := lang.L("days")
		if days == 1 {
			daysStr = lang.L("day")
		}
		str = fmt.Sprintf("%d %s ", days, daysStr)
	}
	if hr > 0 {
		hrStr := lang.L("hrs")
		if hr == 1 {
			hrStr = lang.L("hr")
		}
		str += fmt.Sprintf("%d %s ", hr, hrStr)
	}
	if min > 0 {
		str += fmt.Sprintf("%d %s ", min, lang.L("min"))
	}
	if sec > 0 {
		str += fmt.Sprintf("%d %s ", sec, lang.L("sec"))
	}
	return str[:len(str)-1]
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
	baseType := lang.L("Album")
	switch {
	case releaseTypes&mediaprovider.ReleaseTypeAudiobook > 0:
		baseType = lang.L("Audiobook")
	case releaseTypes&mediaprovider.ReleaseTypeAudioDrama > 0:
		baseType = lang.L("Audio Drama")
	case releaseTypes&mediaprovider.ReleaseTypeBroadcast > 0:
		baseType = lang.L("Broadcast")
	case releaseTypes&mediaprovider.ReleaseTypeDJMix > 0:
		baseType = lang.L("DJ-Mix")
	case releaseTypes&mediaprovider.ReleaseTypeEP > 0:
		baseType = lang.L("EP")
	case releaseTypes&mediaprovider.ReleaseTypeFieldRecording > 0:
		baseType = lang.L("Field Recording")
	case releaseTypes&mediaprovider.ReleaseTypeInterview > 0:
		baseType = lang.L("Interview")
	case releaseTypes&mediaprovider.ReleaseTypeMixtape > 0:
		baseType = lang.L("Mixtape")
	case releaseTypes&mediaprovider.ReleaseTypeSingle > 0:
		baseType = lang.L("Single")
	case releaseTypes&mediaprovider.ReleaseTypeSoundtrack > 0:
		baseType = lang.L("Soundtrack")
	}

	var modifiers []string
	if releaseTypes&mediaprovider.ReleaseTypeLive > 0 {
		modifiers = append(modifiers, lang.L("Live"))
	}
	if releaseTypes&mediaprovider.ReleaseTypeDemo > 0 {
		modifiers = append(modifiers, lang.L("Demo"))
	}
	if releaseTypes&mediaprovider.ReleaseTypeRemix > 0 {
		modifiers = append(modifiers, lang.L("Remix"))
	}
	if releaseTypes&mediaprovider.ReleaseTypeSpokenWord > 0 {
		modifiers = append(modifiers, lang.L("Spoken Word"))
	}
	if releaseTypes&mediaprovider.ReleaseTypeCompilation > 0 {
		modifiers = append(modifiers, lang.L("Compilation"))
	}

	modifiers = append(modifiers, baseType)
	return strings.Join(modifiers, " ")
}

func NewRatingSubmenu(onSetRating func(int)) *fyne.MenuItem {
	newRatingMenuItem := func(rating int) *fyne.MenuItem {
		label := fmt.Sprintf("(%s)", lang.L("none"))
		if rating > 0 {
			label = strconv.Itoa(rating)
		}
		return fyne.NewMenuItem(label, func() {
			onSetRating(rating)
		})
	}
	ratingMenu := fyne.NewMenuItem(lang.L("Set rating"), nil)
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

func AddHeaderBackground(obj fyne.CanvasObject) *fyne.Container {
	return AddHeaderBackgroundWithColorName(obj, myTheme.ColorNamePageHeader)
}

func AddHeaderBackgroundWithColorName(obj fyne.CanvasObject, colorName fyne.ThemeColorName) *fyne.Container {
	bgrnd := myTheme.NewThemedRectangle(colorName)
	bgrnd.CornerRadiusName = theme.SizeNameInputRadius
	return container.NewStack(bgrnd,
		container.New(&layout.CustomPaddedLayout{LeftPadding: 10, RightPadding: 10, TopPadding: 10, BottomPadding: 10},
			obj))
}

func NewTruncatingRichText() *widget.RichText {
	rt := widget.NewRichTextWithText("")
	rt.Truncation = fyne.TextTruncateEllipsis
	return rt
}

func NewTruncatingLabel() *widget.Label {
	rt := widget.NewLabel("")
	rt.Truncation = fyne.TextTruncateEllipsis
	return rt
}

func NewTrailingAlignLabel() *widget.Label {
	rt := widget.NewLabel("")
	rt.Alignment = fyne.TextAlignTrailing
	return rt
}

func SaveWindowSize(w fyne.Window, wPtr, hPtr *int) {
	// round sizes to even to avoid Wayland issues with 2x scaling factor
	// https://github.com/dweymouth/supersonic/issues/212
	*wPtr = int(math.RoundToEven(float64(w.Canvas().Size().Width)))
	*hPtr = int(math.RoundToEven(float64(w.Canvas().Size().Height)))
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
