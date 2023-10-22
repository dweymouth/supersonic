package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type MultiHyperlink struct {
	widget.BaseWidget

	Segments []MultiHyperlinkSegment
	OnTapped func(string)

	minSegWidthCached float32
	minHeightCached   float32
	separatorWCached  float32

	// TODO: Once https://github.com/fyne-io/fyne/issues/4336 is resolved,
	//   we can switch to the much cleaner RichText implementation
	//provider *widget.RichText
	content *fyne.Container
}

type MultiHyperlinkSegment struct {
	Text   string
	LinkID string
}

func NewMultiHyperlink() *MultiHyperlink {
	c := &MultiHyperlink{
		//provider: widget.NewRichText(),
		content: container.NewWithoutLayout(),
	}
	c.ExtendBaseWidget(c)
	//c.provider.Wrapping = fyne.TextTruncate
	return c
}

func (c *MultiHyperlink) getMinSegWidth() float32 {
	if c.minSegWidthCached == 0 {
		c.minSegWidthCached = fyne.MeasureText(", W", theme.TextSize(), fyne.TextStyle{}).Width
	}
	return c.minSegWidthCached
}

func (c *MultiHyperlink) getSeparatorWidth() float32 {
	if c.separatorWCached == 0 {
		c.separatorWCached = fyne.MeasureText(",", theme.TextSize(), fyne.TextStyle{}).Width
	}
	return c.separatorWCached
}

func (c *MultiHyperlink) layoutObjects() {
	if len(c.Segments) == 0 {
		c.content.RemoveAll()
		return
	}
	x := float32(0)
	width := c.Size().Width
	l := len(c.content.Objects)

	var i int // at end of loop should be index of last seg that was laid out for display
	var seg MultiHyperlinkSegment
	for i, seg = range c.Segments {
		if i > 0 {
			// check if we have enough room to show another segment
			if x+c.getMinSegWidth() > width {
				i -= 1 // we are not showing this segment
				break
			}
		}

		appendingSegments := 2*i >= l
		var obj fyne.CanvasObject
		if !appendingSegments {
			obj = c.content.Objects[2*i]
		} else if i > 0 {
			c.content.Objects = append(c.content.Objects, c.newSeparatorLabel())
		}
		if seg.LinkID == "" {
			obj = c.updateOrReplaceLabel(obj, seg.Text)
		} else {
			obj = c.updateOrReplaceHyperlink(obj, seg.Text)
		}
		if appendingSegments {
			c.content.Objects = append(c.content.Objects, obj)
		} else {
			c.content.Objects[2*i] = obj
		}

		if i > 0 {
			// move and resize separator
			obj = c.content.Objects[2*i-1]
			ms := obj.MinSize()
			obj.Resize(ms)
			obj.Move(fyne.NewPos(x-ms.Width+c.getSeparatorWidth()+1, 0)) // this is really ugly
			x -= theme.Padding() * 2
		}
		// move and resize text object
		textW := fyne.MeasureText(seg.Text, theme.TextSize(), fyne.TextStyle{}).Width + theme.Padding()*2 + theme.InnerPadding()
		obj = c.content.Objects[2*i]
		ms := obj.MinSize()
		w := fyne.Min(width-x, textW)
		obj.Resize(fyne.NewSize(w, ms.Height))
		obj.Move(fyne.NewPos(x, 0))
		x += w
	}

	i += 1
	c.content.Objects = c.content.Objects[:2*i-1]
}

func (c *MultiHyperlink) updateOrReplaceLabel(obj fyne.CanvasObject, text string) fyne.CanvasObject {
	if obj != nil {
		if label, ok := obj.(*widget.Label); ok {
			label.Text = text
			return label
		}
	}
	l := widget.NewLabel(text)
	l.Wrapping = fyne.TextTruncate
	return l
}

func (c *MultiHyperlink) updateOrReplaceHyperlink(obj fyne.CanvasObject, text string) fyne.CanvasObject {
	if obj != nil {
		if link, ok := obj.(*widget.Hyperlink); ok {
			link.Text = text
			return link
		}
	}
	l := widget.NewHyperlink(text, nil)
	l.Wrapping = fyne.TextTruncate
	return l
}

func (c *MultiHyperlink) newSeparatorLabel() *widget.Label {
	return widget.NewLabel(", ")
}

/***
 * RichText implementation

func (c *MultiHyperlink) syncSegments() {
	l := len(c.provider.Segments)
	for i, seg := range c.Segments {
		appendingSegments := 2*i >= l // true if we need to extend the RichText provider with new segments
		var rtSeg widget.RichTextSegment
		if !appendingSegments {
			rtSeg = c.provider.Segments[2*i]
		} else if i > 0 {
			// append new separator segment
			c.provider.Segments = append(c.provider.Segments, c.newSeparatorSegment())
		}
		if seg.LinkID == "" {
			rtSeg = c.updateOrReplaceTextSegment(rtSeg, seg.Text)
		} else {
			rtSeg = c.updateOrReplaceHyperlinkSegment(rtSeg, seg.Text, seg.LinkID)
		}
		if appendingSegments {
			c.provider.Segments = append(c.provider.Segments, rtSeg)
		} else {
			c.provider.Segments[2*i] = rtSeg
		}
	}
	// discard extra segments if shortening the multihyperlink
	for i := 2 * len(c.Segments); i < l; i++ {
		c.provider.Segments[i] = nil
	}
	c.provider.Segments = c.provider.Segments[:2*len(c.Segments)-1]
}

func (c *MultiHyperlink) newSeparatorSegment() widget.RichTextSegment {
	return &widget.TextSegment{Text: ", ", Style: widget.RichTextStyle{Inline: true}}
}

func (c *MultiHyperlink) updateOrReplaceTextSegment(seg widget.RichTextSegment, text string) widget.RichTextSegment {
	if seg != nil {
		if ts, ok := seg.(*widget.TextSegment); ok {
			ts.Text = text
			return seg
		}
	}
	return &widget.TextSegment{Text: text, Style: widget.RichTextStyle{Inline: true}}
}

func (c *MultiHyperlink) updateOrReplaceHyperlinkSegment(seg widget.RichTextSegment, text, linkID string) widget.RichTextSegment {
	if seg != nil {
		if ts, ok := seg.(*widget.HyperlinkSegment); ok {
			ts.Text = text
			// TODO: OnTapped
			return seg
		}
	}
	return &widget.HyperlinkSegment{Text: text} // TODO: OnTapped
}

*/

func (c *MultiHyperlink) onSegmentTapped(linkID string) {
	// TODO
}

func (c *MultiHyperlink) MinSize() fyne.Size {
	if c.minHeightCached == 0 {
		c.minHeightCached = widget.NewLabel("").MinSize().Height
	}
	return fyne.NewSize(1, c.minHeightCached)
}

func (c *MultiHyperlink) Refresh() {
	//c.syncSegments()
	c.layoutObjects()
	c.BaseWidget.Refresh()
}

func (c *MultiHyperlink) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(c.content)
	//return widget.NewSimpleRenderer(c.provider)
}
