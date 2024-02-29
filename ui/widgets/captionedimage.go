package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

type Aspectable interface {
	fyne.CanvasObject
	Aspect() float32
}

type CaptionedImage struct {
	widget.BaseWidget

	Content Aspectable
	Caption fyne.CanvasObject
}

var _ fyne.Widget = (*CaptionedImage)(nil)

func NewCaptionedImage(content Aspectable, caption fyne.CanvasObject) *CaptionedImage {
	c := &CaptionedImage{Content: content, Caption: caption}
	c.ExtendBaseWidget(c)
	return c
}

func (c *CaptionedImage) CreateRenderer() fyne.WidgetRenderer {
	return &captionedImageRenderer{
		ci: c,
	}
}

type captionedImageRenderer struct {
	ci *CaptionedImage
}

var _ fyne.WidgetRenderer = (*captionedImageRenderer)(nil)

func (*captionedImageRenderer) Destroy() {
	// intentionally blank
}

func (c *captionedImageRenderer) Layout(s fyne.Size) {
	content := c.ci.Content
	aspect := content.Aspect()
	caption := c.ci.Caption
	var captionHeight float32
	if caption != nil {
		captionHeight = caption.MinSize().Height
	}

	// max height that could be allocated to Content
	contentMaxedHeight := s.Height
	if caption != nil {
		contentMaxedHeight -= captionHeight
	}

	// aspect ratio of Content if it were maxed out
	maxedAspect := s.Width / contentMaxedHeight
	if maxedAspect > aspect {
		// Will use full height, but not full width
		width := contentMaxedHeight * aspect
		content.Resize(fyne.NewSize(width, contentMaxedHeight))
		xStart := (s.Width - width) / 2
		content.Move(fyne.NewPos(xStart, 0))
		if caption != nil {
			caption.Resize(fyne.NewSize(width, captionHeight))
			caption.Move(fyne.NewPos(xStart, contentMaxedHeight))
		}
		return
	}

	// Content will not use full height
	// Positioning of Content and Caption will be adjusted to center
	contentH := s.Width / aspect
	yStart := (s.Height - contentH - captionHeight) / 2
	content.Resize(fyne.NewSize(s.Width, contentH))
	content.Move(fyne.NewPos(0, yStart))
	if caption != nil {
		caption.Resize(fyne.NewSize(s.Width, captionHeight))
		caption.Move(fyne.NewPos(0, yStart+contentH))
	}
}

func (c *captionedImageRenderer) MinSize() fyne.Size {
	objSize := c.ci.Content.MinSize()
	if c.ci.Caption == nil {
		return objSize
	}
	cptSize := c.ci.Caption.MinSize()
	return fyne.NewSize(
		fyne.Max(objSize.Width, cptSize.Width),
		objSize.Height+cptSize.Height,
	)
}

func (c *captionedImageRenderer) Objects() []fyne.CanvasObject {
	if c.ci.Caption != nil {
		return []fyne.CanvasObject{c.ci.Content, c.ci.Caption}
	}
	return []fyne.CanvasObject{c.ci.Content}
}

func (c *captionedImageRenderer) Refresh() {
	c.ci.Content.Refresh()
	if cap := c.ci.Caption; cap != nil {
		cap.Refresh()
	}
	// aspect may have changed on Content,
	// so we need to re-layout
	c.Layout(c.ci.Size())
}
