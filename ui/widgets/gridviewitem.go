package widgets

import (
	"image"

	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/res"
	"github.com/dweymouth/supersonic/sharedutil"
	"github.com/dweymouth/supersonic/ui/layouts"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

var _ fyne.Widget = (*GridViewItem)(nil)

var _ fyne.Widget = (*coverImage)(nil)

type coverImage struct {
	widget.BaseWidget

	Im                *ImagePlaceholder
	playbtn           *canvas.Image
	mouseInsideBtn    bool
	OnPlay            func()
	OnShowPage        func()
	OnShowContextMenu func(fyne.Position)
}

var (
	playBtnSize        = fyne.NewSize(60, 60)
	playBtnHoveredSize = fyne.NewSize(65, 65)
)

func newCoverImage(placeholderResource fyne.Resource) *coverImage {
	c := &coverImage{}
	c.Im = NewImagePlaceholder(placeholderResource, 200)
	c.Im.OnTapped = c.Tapped
	c.Im.OnTappedSecondary = c.TappedSecondary
	c.Im.ScaleMode = canvas.ImageScaleFastest
	c.playbtn = &canvas.Image{FillMode: canvas.ImageFillContain, Resource: res.ResPlaybuttonPng}
	c.playbtn.SetMinSize(playBtnSize)
	c.playbtn.Hidden = true
	c.ExtendBaseWidget(c)
	return c
}

func (c *coverImage) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(
		container.NewStack(c.Im, container.NewCenter(c.playbtn)),
	)
}

func (c *coverImage) Cursor() desktop.Cursor {
	return desktop.PointerCursor
}

func (c *coverImage) Tapped(e *fyne.PointEvent) {
	if isInside(c.center(), c.playbtn.Size().Height/2, e.Position) {
		if c.OnPlay != nil {
			c.OnPlay()
		}
		return
	}
	if c.OnShowPage != nil {
		c.OnShowPage()
	}
}

func (c *coverImage) TappedSecondary(e *fyne.PointEvent) {
	if c.OnShowContextMenu != nil {
		c.OnShowContextMenu(e.AbsolutePosition)
	}
}

func (a *coverImage) MouseIn(*desktop.MouseEvent) {
	a.playbtn.Hidden = false
	a.Refresh()
}

func (a *coverImage) MouseOut() {
	a.mouseInsideBtn = false
	a.playbtn.Hidden = true
	a.Refresh()
}

func (a *coverImage) MouseMoved(e *desktop.MouseEvent) {
	if isInside(a.center(), a.playbtn.MinSize().Height/2, e.Position) {
		if !a.mouseInsideBtn {
			a.playbtn.SetMinSize(playBtnHoveredSize)
			a.playbtn.Refresh()
		}
		a.mouseInsideBtn = true
	} else {
		if a.mouseInsideBtn {
			a.playbtn.SetMinSize(playBtnSize)
			a.playbtn.Refresh()
		}
		a.mouseInsideBtn = false
	}
}

func (a *coverImage) center() fyne.Position {
	return fyne.NewPos(a.Size().Width/2, a.Size().Height/2)
}

func (a *coverImage) SetImage(im image.Image) {
	a.Im.SetImage(im, true)
}

func (a *coverImage) ResetPlayButton() {
	a.playbtn.SetMinSize(playBtnSize)
	a.mouseInsideBtn = false
	a.playbtn.Hidden = true
}

func isInside(origin fyne.Position, radius float32, point fyne.Position) bool {
	x, y := (point.X - origin.X), (point.Y - origin.Y)
	return x*x+y*y <= radius*radius
}

type GridViewItemModel struct {
	Name         string
	ID           string
	CoverArtID   string
	Secondary    []string
	SecondaryIDs []string
}

type GridViewItem struct {
	widget.BaseWidget

	itemID        string
	secondaryIDs  []string
	primaryText   *widget.Hyperlink
	secondaryText *MultiHyperlink
	container     *fyne.Container

	// updated by GridView
	Cover     *coverImage
	ImgLoader backend.ThumbnailLoader

	OnPlay              func()
	OnShowContextMenu   func(fyne.Position)
	OnShowItemPage      func()
	OnShowSecondaryPage func(string)
}

func NewGridViewItem(placeholderResource fyne.Resource) *GridViewItem {
	g := &GridViewItem{
		primaryText:   widget.NewHyperlink("", nil),
		secondaryText: NewMultiHyperlink(),
		Cover:         newCoverImage(placeholderResource),
	}
	g.primaryText.TextStyle.Bold = true
	g.primaryText.Truncation = fyne.TextTruncateEllipsis
	g.ExtendBaseWidget(g)
	g.Cover.OnPlay = func() {
		if g.OnPlay != nil {
			g.OnPlay()
		}
	}
	g.Cover.OnShowContextMenu = func(pos fyne.Position) {
		if g.OnShowContextMenu != nil {
			g.OnShowContextMenu(pos)
		}
	}
	showItemFn := func() {
		if g.OnShowItemPage != nil {
			g.OnShowItemPage()
		}
	}
	g.Cover.OnShowPage = showItemFn
	g.primaryText.OnTapped = showItemFn
	g.secondaryText.OnTapped = func(s string) {
		if g.OnShowSecondaryPage != nil {
			g.OnShowSecondaryPage(s)
		}
	}

	g.createContainer()
	return g
}

func (g *GridViewItem) createContainer() {
	info := container.New(&layouts.VboxCustomPadding{ExtraPad: -16}, g.primaryText, g.secondaryText)
	c := container.New(&layouts.VboxCustomPadding{ExtraPad: -5}, g.Cover, info)
	pad := &layouts.CenterPadLayout{PadLeftRight: 20, PadTopBottom: 10}
	g.container = container.New(pad, c)
}

func (g *GridViewItem) NeedsUpdate(model GridViewItemModel) bool {
	return g.itemID != model.ID || !sharedutil.SliceEqual(g.secondaryIDs, model.SecondaryIDs)
}

func (g *GridViewItem) Update(model GridViewItemModel) {
	g.itemID = model.ID
	g.secondaryIDs = model.SecondaryIDs
	g.primaryText.SetText(model.Name)
	g.secondaryText.BuildSegments(model.Secondary, model.SecondaryIDs)
	g.secondaryText.Refresh()
	g.Cover.ResetPlayButton()
}

func (g *GridViewItem) Refresh() {
	g.BaseWidget.Refresh()
}

func (g *GridViewItem) ItemID() string {
	return g.itemID
}

func (g *GridViewItem) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(g.container)
}

// steal hover events from underlying GridWrap to prevent unwanted
// hover rectangle backgrounds
var _ desktop.Hoverable = (*GridViewItem)(nil)

func (g *GridViewItem) MouseIn(e *desktop.MouseEvent) {}

func (g *GridViewItem) MouseOut() {}

func (g *GridViewItem) MouseMoved(e *desktop.MouseEvent) {}
