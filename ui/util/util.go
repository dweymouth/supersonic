package util

import (
	"fmt"
	"image"
	"math"
	"strings"

	"fyne.io/fyne/v2"
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

func ImageAspect(im image.Image) float32 {
	b := im.Bounds()
	return float32(b.Max.X-b.Min.X) / float32(b.Max.Y-b.Min.Y)
}

type PopUpProvider interface {
	CreatePopUp(fyne.CanvasObject) *widget.PopUp
	WindowSize() fyne.Size
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
