package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

type MultiHyperlink struct {
	widget.BaseWidget

	Segments []MultiHyperlinkSegment
	OnTapped func(string)

	provider *widget.RichText
}

type MultiHyperlinkSegment struct {
	Text   string
	LinkID string
}

func NewMultiHyperlink() *MultiHyperlink {
	c := &MultiHyperlink{
		provider: widget.NewRichText(),
	}
	c.ExtendBaseWidget(c)
	c.provider.Wrapping = fyne.TextTruncate
	return c
}

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

func (c *MultiHyperlink) onSegmentTapped(linkID string) {
	// TODO
}

func (c *MultiHyperlink) Refresh() {
	c.syncSegments()
	c.BaseWidget.Refresh()
}

func (c *MultiHyperlink) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(c.provider)
}
