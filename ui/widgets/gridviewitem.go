package widgets

import (
	"image"
	"image/color"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	ttwidget "github.com/dweymouth/fyne-tooltip/widget"
	"github.com/dweymouth/supersonic/res"
	myTheme "github.com/dweymouth/supersonic/ui/theme"
	"github.com/dweymouth/supersonic/ui/util"
)

var _ fyne.Widget = (*GridViewItem)(nil)
var _ fyne.Focusable = (*GridViewItem)(nil)

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
	Suffix       string
}

type GridViewItem struct {
	widget.BaseWidget

	ShowSuffix bool

	itemID        string
	secondaryIDs  []string
	primaryText   *ttwidget.Hyperlink
	secondaryText *MultiHyperlink
	suffix        string
	container     *fyne.Container
	focused       bool
	focusRect     *canvas.Rectangle

	// updated by GridView
	Cover     *coverImage
	ImgLoader util.ThumbnailLoader
	ItemIndex int

	OnPlay              func()
	OnShowContextMenu   func(fyne.Position)
	OnShowItemPage      func()
	OnShowSecondaryPage func(string)

	// Invoked with arg 0-3 when left, right, up, or down neighbor should be focused, respectively
	OnFocusNeighbor func(int)
}

func NewGridViewItem(placeholderResource fyne.Resource) *GridViewItem {
	g := &GridViewItem{
		primaryText:   ttwidget.NewHyperlink("", nil),
		secondaryText: NewMultiHyperlink(),
		Cover:         newCoverImage(placeholderResource),
	}
	g.primaryText.Truncation = fyne.TextTruncateEllipsis
	g.primaryText.TextStyle.Bold = true
	g.secondaryText.SizeName = myTheme.SizeNameSubText
	g.secondaryText.SuffixSizeName = myTheme.SizeNameSuffixText
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
	info := container.New(layout.NewCustomPaddedVBoxLayout(theme.Padding()-17), g.primaryText, g.secondaryText)
	g.focusRect = canvas.NewRectangle(color.Transparent)
	g.focusRect.StrokeWidth = 3
	coverStack := container.NewStack(g.Cover, g.focusRect)
	c := container.New(layout.NewCustomPaddedVBoxLayout(theme.Padding()-5), coverStack, info)
	pad := layout.NewCustomPaddedLayout(10, 10, 20, 20)
	g.container = container.New(pad, c)
}

func (g *GridViewItem) NeedsUpdate(model GridViewItemModel) bool {
	return g.itemID != model.ID || !slices.Equal(g.secondaryIDs, model.SecondaryIDs) ||
		g.secondaryText.Suffix != model.Suffix
}

func (g *GridViewItem) Update(model GridViewItemModel) {
	g.itemID = model.ID
	g.secondaryIDs = model.SecondaryIDs
	g.primaryText.SetText(model.Name)
	g.primaryText.SetToolTip(model.Name)
	g.secondaryText.BuildSegments(model.Secondary, model.SecondaryIDs)
	if g.ShowSuffix {
		g.secondaryText.Suffix = model.Suffix
	} else {
		g.secondaryText.Suffix = ""
	}
	g.secondaryText.Refresh()
	g.Cover.ResetPlayButton()
	if g.focused {
		fyne.CurrentApp().Driver().CanvasForObject(g).Focus(nil)
		g.FocusLost()
	}
}

func (g *GridViewItem) Refresh() {
	if g.ShowSuffix && g.secondaryText.Suffix == "" && g.suffix != "" {
		g.secondaryText.Suffix = g.suffix
		g.secondaryText.Refresh()
	} else if !g.ShowSuffix && g.secondaryText.Suffix != "" {
		g.secondaryText.Suffix = ""
		g.secondaryText.Refresh()
	}
	g.focusRect.StrokeColor = util.MakeOpaque(theme.FocusColor())
	g.focusRect.Hidden = !g.focused
}

func (g *GridViewItem) ItemID() string {
	return g.itemID
}

func (g *GridViewItem) FocusGained() {
	g.focused = true
	g.Refresh()
}

func (g *GridViewItem) FocusLost() {
	g.focused = false
	g.Refresh()
}

func (g *GridViewItem) TypedKey(e *fyne.KeyEvent) {
	if !g.focused {
		return
	}
	focusArg := -1
	switch e.Name {
	case fyne.KeyLeft:
		focusArg = 0
	case fyne.KeyRight:
		focusArg = 1
	case fyne.KeyUp:
		focusArg = 2
	case fyne.KeyDown:
		focusArg = 3
	case fyne.KeyEnter:
		fallthrough
	case fyne.KeyReturn:
		fallthrough
	case fyne.KeySpace:
		if g.OnShowItemPage != nil {
			g.OnShowItemPage()
			return
		}
	}
	if focusArg >= 0 && g.OnFocusNeighbor != nil {
		g.OnFocusNeighbor(focusArg)
	}
}

func (g *GridViewItem) TypedRune(rune) {
	// intentionally blank
}

var _ fyne.Tappable = (*GridViewItem)(nil)

func (g *GridViewItem) Tapped(*fyne.PointEvent) {
	fyne.CurrentApp().Driver().CanvasForObject(g).Unfocus()
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
