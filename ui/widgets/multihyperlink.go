package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"
)

type MultiHyperlink struct {
	widget.BaseWidget

	Segments []MultiHyperlinkSegment

	// Suffix string that is appended (with · separator, or parenthesized
	// if SuffixParenthesized is true) only if there is enough room
	Suffix string

	SuffixParenthesized bool

	OnMouseIn  func(*desktop.MouseEvent)
	OnMouseOut func()
	OnTapped   func(string)

	SizeName       fyne.ThemeSizeName
	SuffixSizeName fyne.ThemeSizeName

	minSegWidthCached float32
	minHeightCached   float32
	separatorWCached  float32

	// TODO: Once https://github.com/fyne-io/fyne/issues/4336 is resolved,
	//   we can switch to the much cleaner RichText implementation
	// provider *widget.RichText

	objects     []fyne.CanvasObject
	suffixLabel *ttwidget.RichText
	content     *fyne.Container
}

type MultiHyperlinkSegment struct {
	Text   string
	LinkID string
}

func NewMultiHyperlink() *MultiHyperlink {
	c := &MultiHyperlink{
		// provider: widget.NewRichText(),
		content: container.NewWithoutLayout(),
	}
	c.ExtendBaseWidget(c)
	// c.provider.Truncation = fyne.TextTruncateEllipsis
	return c
}

func (m *MultiHyperlink) BuildSegments(texts, links []string) {
	l := len(links)
	m.Segments = nil
	for i, text := range texts {
		link := ""
		if l > i {
			link = links[i]
		}
		m.Segments = append(m.Segments, MultiHyperlinkSegment{Text: text, LinkID: link})
	}
}

func (c *MultiHyperlink) getMinSegWidth() float32 {
	if c.minSegWidthCached == 0 {
		c.minSegWidthCached = fyne.MeasureText(", W", theme.Size(c.sizeName()), fyne.TextStyle{}).Width
	}
	return c.minSegWidthCached
}

func (c *MultiHyperlink) getSeparatorWidth() float32 {
	if c.separatorWCached == 0 {
		c.separatorWCached = fyne.MeasureText(",", theme.Size(c.sizeName()), fyne.TextStyle{}).Width
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
	l := len(c.objects)

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
			obj = c.objects[2*i]
		} else if i > 0 {
			c.objects = append(c.objects, c.newSeparatorLabel())
		}
		if seg.LinkID == "" {
			obj = c.updateOrReplaceLabel(obj, seg.Text)
		} else {
			obj = c.updateOrReplaceHyperlink(obj, seg.Text, seg.LinkID)
		}
		if appendingSegments {
			c.objects = append(c.objects, obj)
		} else {
			c.objects[2*i] = obj
		}

		if i > 0 {
			// move and resize separator
			obj = c.objects[2*i-1]
			ms := obj.MinSize()
			obj.Resize(ms)
			obj.Move(fyne.NewPos(x-ms.Width+c.getSeparatorWidth()+1, 0)) // this is really ugly
			x -= theme.Padding() * 2
		}
		// move and resize text object
		// extra +3 to textW gives it just enough space to not trigger ellipsis truncation
		textW := fyne.MeasureText(seg.Text, theme.Size(c.sizeName()), fyne.TextStyle{}).Width + theme.Padding()*2 + theme.InnerPadding() + 3
		obj = c.objects[2*i]
		ms := obj.MinSize()
		w := fyne.Min(width-x, textW)
		obj.Resize(fyne.NewSize(w, ms.Height))
		obj.Move(fyne.NewPos(x, 0))
		x += w
	}

	i += 1
	c.content.Objects = c.objects[:2*i-1]
	if i == len(c.Segments) && c.Suffix != "" {
		var suffixText string
		if c.SuffixParenthesized {
			suffixText = "\u2009(" + c.Suffix + ")"
		} else {
			suffixText = "· " + c.Suffix
		}
		if c.suffixLabel == nil {
			c.suffixLabel = ttwidget.NewRichTextWithText(suffixText)
			c.suffixLabel.OnMouseIn = c.callOnMouseIn
			c.suffixLabel.OnMouseOut = c.callOnMouseOut
		} else {
			c.suffixLabel.Segments[0].(*widget.TextSegment).Text = suffixText
		}
		sizeName := c.sizeName()
		if c.SuffixSizeName != "" {
			sizeName = c.SuffixSizeName
		}
		// TODO: the magic numbers to get exact positioning here are gross
		c.suffixLabel.Segments[0].(*widget.TextSegment).Style.SizeName = sizeName
		innerPad2 := theme.InnerPadding() * 2
		if x+c.suffixLabel.MinSize().Width-innerPad2*1.3 < width {
			y := theme.Size(c.sizeName()) - theme.Size(sizeName)
			c.suffixLabel.Move(fyne.NewPos(x-innerPad2+1, y))
			c.content.Objects = append(c.content.Objects, c.suffixLabel)
		}
	}
}

func (c *MultiHyperlink) updateOrReplaceLabel(obj fyne.CanvasObject, text string) fyne.CanvasObject {
	if obj != nil {
		if label, ok := obj.(*ttwidget.RichText); ok {
			ts := label.Segments[0].(*widget.TextSegment)
			ts.Text = text
			ts.Style.SizeName = c.sizeName()
			label.SetToolTip(text)
			return label
		}
	}
	l := ttwidget.NewRichTextWithText(text)
	l.SetToolTip(text)
	l.OnMouseIn = c.callOnMouseIn
	l.OnMouseOut = c.callOnMouseOut
	l.Segments[0].(*widget.TextSegment).Style.SizeName = c.sizeName()
	l.Truncation = fyne.TextTruncateEllipsis
	return l
}

func (c *MultiHyperlink) updateOrReplaceHyperlink(obj fyne.CanvasObject, text, link string) fyne.CanvasObject {
	if obj != nil {
		if l, ok := obj.(*ttwidget.Hyperlink); ok {
			l.Text = text
			l.SetToolTip(text)
			l.SizeName = c.sizeName()
			l.OnTapped = func() { c.onSegmentTapped(link) }
			return l
		}
	}
	l := ttwidget.NewHyperlink(text, nil)
	l.SetToolTip(text)
	l.SizeName = c.sizeName()
	l.Truncation = fyne.TextTruncateEllipsis
	l.OnTapped = func() { c.onSegmentTapped(link) }
	return l
}

func (c *MultiHyperlink) callOnMouseIn(e *desktop.MouseEvent) {
	if f := c.OnMouseIn; f != nil {
		f(e)
	}
}

func (c *MultiHyperlink) callOnMouseOut() {
	if f := c.OnMouseOut; f != nil {
		f()
	}
}

func (c *MultiHyperlink) newSeparatorLabel() *widget.RichText {
	rt := widget.NewRichTextWithText(", ")
	rt.Segments[0].(*widget.TextSegment).Style.SizeName = c.sizeName()
	return rt
}

func (c *MultiHyperlink) sizeName() fyne.ThemeSizeName {
	if c.SizeName == "" {
		return theme.SizeNameText
	}
	return c.SizeName
}

/***
 * RichText implementation
 * TODO: add support for SizeName, suffix

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
	if c.OnTapped != nil {
		c.OnTapped(linkID)
	}
}

func (c *MultiHyperlink) MinSize() fyne.Size {
	if c.minHeightCached == 0 {
		c.minHeightCached = widget.NewLabel("").MinSize().Height
	}
	return fyne.NewSize(1, c.minHeightCached)
}

func (c *MultiHyperlink) Resize(size fyne.Size) {
	c.BaseWidget.Resize(size)
	c.layoutObjects()
}

func (c *MultiHyperlink) Refresh() {
	// c.syncSegments()
	c.layoutObjects()
	c.BaseWidget.Refresh()
}

func (c *MultiHyperlink) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(c.content)
	// return widget.NewSimpleRenderer(c.provider)
}
