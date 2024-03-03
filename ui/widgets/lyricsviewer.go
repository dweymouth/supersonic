package widgets

import "fyne.io/fyne/v2/widget"

type LyricsViewer struct {
	widget.Label
}

func NewLyricsViewer() *LyricsViewer {
	l := &LyricsViewer{Label: widget.Label{
		Text: "Lyrics not available",
	}}
	l.ExtendBaseWidget(l)
	return l
}
