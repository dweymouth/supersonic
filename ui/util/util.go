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
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/net/html"
)

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

func RichTextSegsFromHTMLString(s string) []widget.RichTextSegment {
	tokr := html.NewTokenizer(strings.NewReader(s))
	var segs []widget.RichTextSegment

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
				segs = append(segs, &widget.TextSegment{Text: t.Data})
			}
		}
	}

	return segs
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
