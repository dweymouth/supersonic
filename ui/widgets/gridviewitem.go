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

	EnableFavorite bool
	IsFavorite     bool

	OnPlay            func()
	OnFavorite        func(bool)
	OnShowPage        func()
	OnShowContextMenu func(fyne.Position)

	Im              *ImagePlaceholder
	playbtn         *canvas.Image
	favoriteButton  *canvas.Image
	moreButton      *canvas.Image
	prevTheme       fyne.ThemeVariant
	bottomPanel     *fyne.Container
	mouseInsidePlay bool
	mouseInsideFav  bool
	mouseInsideMore bool
}

var (
	playBtnSize        = fyne.NewSize(60, 60)
	playBtnHoveredSize = fyne.NewSize(65, 65)

	resourcesInitted             bool
	heartFilledResource          fyne.Resource
	heartFilledHoveredResource   fyne.Resource
	heartUnfilledResource        fyne.Resource
	heartUnfilledHoveredResource fyne.Resource
	moreVerticalResource         fyne.Resource
	moreVerticalHoveredResource  fyne.Resource
	inlineIconSize               float32
)

func initResources() {
	if resourcesInitted {
		return
	}
	resourcesInitted = true

	inlineIconSize = fyne.CurrentApp().Settings().Theme().Size(theme.SizeNameInlineIcon)

	// TODO: replace util.ColorizeSVG with Fyne's canvas.ColorizeSVG once
	// https://github.com/fyne-io/fyne/pull/5345 is available in Fyne
	c, _ := util.ColorizeSVG(myTheme.NotFavoriteIcon.Content(), myTheme.GridViewIconColor)
	heartUnfilledResource = fyne.NewStaticResource("gridviewnotfavorite", c)
	c, _ = util.ColorizeSVG(myTheme.NotFavoriteIcon.Content(), myTheme.GridViewHoveredIconColor)
	heartUnfilledHoveredResource = fyne.NewStaticResource("gridviewnotfavorite_hover", c)
	c, _ = util.ColorizeSVG(myTheme.FavoriteIcon.Content(), myTheme.GridViewIconColor)
	heartFilledResource = fyne.NewStaticResource("gridviewfavorite", c)
	c, _ = util.ColorizeSVG(myTheme.FavoriteIcon.Content(), myTheme.GridViewHoveredIconColor)
	heartFilledHoveredResource = fyne.NewStaticResource("gridviewfavorite_hover", c)
	c, _ = util.ColorizeSVG(theme.MoreVerticalIcon().Content(), myTheme.GridViewIconColor)
	moreVerticalResource = fyne.NewStaticResource("gridviewmore", c)
	c, _ = util.ColorizeSVG(theme.MoreVerticalIcon().Content(), myTheme.GridViewHoveredIconColor)
	moreVerticalHoveredResource = fyne.NewStaticResource("gridviewmore_hover", c)
}

func newCoverImage(placeholderResource fyne.Resource) *coverImage {
	initResources()
	c := &coverImage{}
	c.Im = NewImagePlaceholder(placeholderResource, 200)
	c.Im.OnTapped = c.Tapped
	c.Im.OnTappedSecondary = c.TappedSecondary
	c.Im.ScaleMode = canvas.ImageScaleFastest
	c.playbtn = &canvas.Image{FillMode: canvas.ImageFillContain, Resource: res.ResPlaybuttonPng}
	c.playbtn.SetMinSize(playBtnSize)
	c.playbtn.Hidden = true

	c.favoriteButton = canvas.NewImageFromResource(heartUnfilledResource)
	c.favoriteButton.SetMinSize(fyne.NewSquareSize(inlineIconSize))
	c.moreButton = canvas.NewImageFromResource(moreVerticalResource)
	c.moreButton.SetMinSize(fyne.NewSquareSize(inlineIconSize))
	c.bottomPanel = container.NewStack(
		canvas.NewVerticalGradient(color.Transparent, color.Black),
		container.NewVBox(
			layout.NewSpacer(), // keep the HBox pushed down
			container.NewHBox(
				layout.NewSpacer(),
				c.favoriteButton,
				c.moreButton,
				util.NewHSpace(0),
			),
			container.New(
				layout.NewCustomPaddedLayout(0, theme.Padding()*2, 0, 0),
				layout.NewSpacer(),
			),
		),
	)
	c.bottomPanel.Hidden = true

	c.ExtendBaseWidget(c)
	return c
}

func (c *coverImage) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(
		container.NewStack(
			c.Im,
			container.NewGridWithRows(2,
				layout.NewSpacer(),
				c.bottomPanel,
			),
			container.NewCenter(c.playbtn),
		),
	)
}

func (c *coverImage) Cursor() desktop.Cursor {
	if c.mouseInsideFav || c.mouseInsideMore || c.mouseInsidePlay {
		return desktop.DefaultCursor
	}
	return desktop.PointerCursor
}

func (c *coverImage) Tapped(e *fyne.PointEvent) {
	if isInside(c.center(), c.playbtn.Size().Height/2, e.Position) {
		if c.OnPlay != nil {
			c.OnPlay()
		}
	} else if c.mouseInsideFav {
		if c.OnFavorite != nil {
			c.IsFavorite = !c.IsFavorite
			c.updateFavoriteIcon(true)
			c.OnFavorite(c.IsFavorite)
		}
	} else if c.mouseInsideMore {
		c.TappedSecondary(e)
	} else {
		if c.OnShowPage != nil {
			c.OnShowPage()
		}
	}
}

func (c *coverImage) TappedSecondary(e *fyne.PointEvent) {
	if c.OnShowContextMenu != nil {
		c.OnShowContextMenu(e.AbsolutePosition)
	}
}

func (a *coverImage) MouseIn(*desktop.MouseEvent) {
	a.playbtn.Hidden = false
	a.updateFavoriteIcon(false)
	a.favoriteButton.Hidden = !a.EnableFavorite
	a.bottomPanel.Hidden = false
	a.Refresh()
}

func (a *coverImage) MouseOut() {
	a.mouseInsidePlay = false
	a.mouseInsideFav = false
	a.mouseInsideMore = false
	a.playbtn.Hidden = true
	a.bottomPanel.Hidden = true
	a.Refresh()
}

func (a *coverImage) MouseMoved(e *desktop.MouseEvent) {
	updateMouseInsidePlay := func(in bool) {
		if in == a.mouseInsidePlay {
			return
		}
		if in {
			a.playbtn.SetMinSize(playBtnHoveredSize)
		} else {
			a.playbtn.SetMinSize(playBtnSize)
		}
		a.playbtn.Refresh()
		a.mouseInsidePlay = in
	}
	updateMouseInsideFav := func(in bool) {
		if in == a.mouseInsideFav {
			return
		}
		if a.IsFavorite {
			if in {
				a.favoriteButton.Resource = heartFilledHoveredResource
			} else {
				a.favoriteButton.Resource = heartFilledResource
			}
		} else {
			if in {
				a.favoriteButton.Resource = heartUnfilledHoveredResource
			} else {
				a.favoriteButton.Resource = heartUnfilledResource
			}
		}
		a.favoriteButton.Refresh()
		a.mouseInsideFav = in
	}
	updateMouseInsideMore := func(in bool) {
		if in == a.mouseInsideMore {
			return
		}
		if in {
			a.moreButton.Resource = moreVerticalHoveredResource
		} else {
			a.moreButton.Resource = moreVerticalResource
		}
		a.moreButton.Refresh()
		a.mouseInsideMore = in
	}

	pad := theme.Padding()
	overFavBtn := e.Position.Y > a.Size().Height-inlineIconSize-pad*3 &&
		e.Position.X > a.Size().Width-inlineIconSize*2-pad*3 &&
		e.Position.X < a.Size().Height-inlineIconSize-pad
	overMoreBtn := e.Position.Y > a.Size().Height-inlineIconSize-pad*3 &&
		e.Position.X > a.Size().Width-inlineIconSize-pad
	if isInside(a.center(), a.playbtn.MinSize().Height/2, e.Position) {
		updateMouseInsidePlay(true)
		updateMouseInsideFav(false)
		updateMouseInsideMore(false)
	} else if overFavBtn {
		updateMouseInsideFav(true)
		updateMouseInsidePlay(false)
		updateMouseInsideMore(false)
	} else if overMoreBtn {
		updateMouseInsideMore(true)
		updateMouseInsideFav(false)
		updateMouseInsidePlay(false)
	} else {
		updateMouseInsideFav(false)
		updateMouseInsidePlay(false)
		updateMouseInsideMore(false)
	}
}

func (a *coverImage) updateFavoriteIcon(refresh bool) {
	if a.IsFavorite {
		a.favoriteButton.Resource = heartFilledResource
	} else {
		a.favoriteButton.Resource = heartUnfilledResource
	}
	if refresh {
		a.favoriteButton.Refresh()
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
	a.mouseInsidePlay = false
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
	CanFavorite  bool
	IsFavorite   bool
}

type GridViewItem struct {
	widget.BaseWidget

	ShowSuffix bool

	model         *GridViewItemModel
	itemID        string
	secondaryIDs  []string
	primaryText   *ttwidget.Hyperlink
	secondaryText *MultiHyperlink
	suffix        string
	focused       bool
	focusRect     *canvas.Rectangle

	// updated by GridView
	Cover     *coverImage
	ImgLoader util.ThumbnailLoader
	ItemIndex int

	OnPlay              func()
	OnFavorite          func(bool)
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
	g.Cover.OnFavorite = func(fav bool) {
		if g.model != nil {
			g.model.IsFavorite = fav
		}
		if g.OnFavorite != nil {
			g.OnFavorite(fav)
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

	g.focusRect = canvas.NewRectangle(color.Transparent)
	g.focusRect.StrokeColor = util.MakeOpaque(theme.FocusColor())
	g.focusRect.StrokeWidth = 3
	g.focusRect.Hidden = true

	return g
}

func (g *GridViewItem) NeedsUpdate(model GridViewItemModel) bool {
	return g.itemID != model.ID || !slices.Equal(g.secondaryIDs, model.SecondaryIDs) ||
		(g.ShowSuffix && g.secondaryText.Suffix != model.Suffix) ||
		(!g.ShowSuffix && g.secondaryText.Suffix != "")
}

func (g *GridViewItem) Update(model *GridViewItemModel) {
	g.Cover.IsFavorite = model.IsFavorite
	g.Cover.EnableFavorite = model.CanFavorite
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
	if g.focused {
		g.focusRect.Show()
	} else {
		g.focusRect.Hide()
	}

	if g.ShowSuffix && g.secondaryText.Suffix == "" && g.suffix != "" {
		g.secondaryText.Suffix = g.suffix
		g.secondaryText.Refresh()
	} else if !g.ShowSuffix && g.secondaryText.Suffix != "" {
		g.secondaryText.Suffix = ""
		g.secondaryText.Refresh()
	}
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
	info := container.New(layout.NewCustomPaddedVBoxLayout(theme.Padding()-17), g.primaryText, g.secondaryText)
	coverStack := container.NewStack(g.Cover, g.focusRect)
	c := container.New(layout.NewCustomPaddedVBoxLayout(theme.Padding()-5), coverStack, info)
	pad := layout.NewCustomPaddedLayout(10, 10, 20, 20)

	return widget.NewSimpleRenderer(container.New(pad, c))
}

// steal hover events from underlying GridWrap to prevent unwanted
// hover rectangle backgrounds
var _ desktop.Hoverable = (*GridViewItem)(nil)

func (g *GridViewItem) MouseIn(e *desktop.MouseEvent) {}

func (g *GridViewItem) MouseOut() {}

func (g *GridViewItem) MouseMoved(e *desktop.MouseEvent) {}
