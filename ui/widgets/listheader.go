package widgets

import (
	"log"

	"github.com/dweymouth/supersonic/ui/layouts"
	myTheme "github.com/dweymouth/supersonic/ui/theme"
	"github.com/dweymouth/supersonic/ui/util"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type SortType int

const (
	SortNone SortType = iota
	SortAscending
	SortDescending
)

type ListHeaderSort struct {
	ColNumber int
	Type      SortType
}

type ListColumn struct {
	Text             string
	Alignment        fyne.TextAlign
	CanToggleVisible bool
}

type ListHeader struct {
	widget.BaseWidget

	DisableSorting bool

	OnColumnSortChanged         func(ListHeaderSort)
	OnColumnVisibilityChanged   func(int, bool)
	OnColumnVisibilityMenuShown func(*widget.PopUp)

	sort          ListHeaderSort
	columns       []ListColumn
	columnVisible []bool
	columnsLayout *layouts.ColumnsLayout

	columnsContainer *fyne.Container
	container        *fyne.Container
	popUpMenu        *fyne.Container
}

func NewListHeader(cols []ListColumn, layout *layouts.ColumnsLayout) *ListHeader {
	l := &ListHeader{
		columns:          cols,
		columnsLayout:    layout,
		columnsContainer: container.New(layout),
	}
	l.columnVisible = make([]bool, len(cols))
	for i := range l.columnVisible {
		l.columnVisible[i] = true
	}
	l.container = container.NewMax(myTheme.NewThemedRectangle(theme.ColorNameBackground), l.columnsContainer)
	l.ExtendBaseWidget(l)
	l.buildColumns()
	return l
}

func (l *ListHeader) SetColumnVisible(colNum int, visible bool) {
	if colNum >= len(l.columns) {
		log.Println("error: ListHeader.SetColumnVisible: column index out of range")
		return
	}
	if visible {
		l.columnsContainer.Objects[colNum].Show()
	} else {
		l.columnsContainer.Objects[colNum].Hide()
	}
	l.columnVisible[colNum] = visible
	l.columnsContainer.Refresh()
}

func (l *ListHeader) buildColumns() {
	for i, c := range l.columns {
		hdr := newColHeader(c, &l.DisableSorting)
		hdr.OnSortChanged = func(i int) func(SortType) {
			return func(sort SortType) { l.SetSorting(ListHeaderSort{ColNumber: i, Type: sort}) }
		}(i)
		hdr.OnTappedSecondary = l.TappedSecondary
		l.columnsContainer.Add(hdr)
	}
}

// Sets the sorting for the ListHeader. Will invoke
// OnColumnSortChanged if set.
func (l *ListHeader) SetSorting(sort ListHeaderSort) {
	if l.sort == sort {
		return
	}
	l.sort = sort
	for i, c := range l.columnsContainer.Objects {
		if i == sort.ColNumber {
			c.(*colHeader).Sort = sort.Type
		} else {
			c.(*colHeader).Sort = SortNone
		}
	}
	l.Refresh()
	if l.OnColumnSortChanged != nil {
		l.OnColumnSortChanged(sort)
	}
}

func (l *ListHeader) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(l.container)
}

func (l *ListHeader) TappedSecondary(e *fyne.PointEvent) {
	l.setupPopUpMenu()
	if len(l.popUpMenu.Objects) == 0 {
		return
	}
	pop := widget.NewPopUp(l.popUpMenu, fyne.CurrentApp().Driver().CanvasForObject(l))
	pop.ShowAtPosition(e.AbsolutePosition)
	if l.OnColumnVisibilityMenuShown != nil {
		l.OnColumnVisibilityMenuShown(pop)
	}
}

func (l *ListHeader) setupPopUpMenu() {
	if l.popUpMenu == nil {
		l.popUpMenu = container.New(&layouts.VboxCustomPadding{ExtraPad: -10})
		for i, c := range l.columns {
			if c.CanToggleVisible {
				l.popUpMenu.Add(widget.NewCheck(c.Text, l.createOnChangedCallbk(i)))
			}
		}
	}
	objIdx := 0
	for i, col := range l.columns {
		if col.CanToggleVisible {
			l.popUpMenu.Objects[objIdx].(*widget.Check).Checked = l.columnVisible[i]
			objIdx++
		}
	}
}

func (l *ListHeader) createOnChangedCallbk(colNum int) func(bool) {
	return func(val bool) {
		l.columnVisible[colNum] = val
		l.SetColumnVisible(colNum, val)
		if l.OnColumnVisibilityChanged != nil {
			l.OnColumnVisibilityChanged(colNum, val)
		}
	}
}

type colHeader struct {
	widget.BaseWidget

	Sort              SortType
	OnSortChanged     func(SortType)
	OnTappedSecondary func(*fyne.PointEvent)

	sortDisabled *bool
	columnCfg    ListColumn

	label             *widget.RichText
	sortIcon          *widget.Icon
	sortIconNegSpacer fyne.CanvasObject
	container         *fyne.Container
}

func newColHeader(columnCfg ListColumn, sortDisabled *bool) *colHeader {
	c := &colHeader{columnCfg: columnCfg, sortDisabled: sortDisabled}
	c.ExtendBaseWidget(c)

	c.label = widget.NewRichTextWithText(columnCfg.Text)
	c.label.Segments[0].(*widget.TextSegment).Style = widget.RichTextStyle{
		TextStyle: fyne.TextStyle{Bold: true},
		Alignment: columnCfg.Alignment,
	}
	c.sortIcon = widget.NewIcon(theme.MenuDropDownIcon())
	// hack to remove extra icon space
	// should be hidden whenever sortIcon is hidden
	c.sortIconNegSpacer = util.NewHSpace(0)

	return c
}

func (c *colHeader) Tapped(*fyne.PointEvent) {
	if *c.sortDisabled {
		return
	}
	switch c.Sort {
	case SortNone:
		c.Sort = SortAscending
	case SortAscending:
		c.Sort = SortDescending
	case SortDescending:
		c.Sort = SortNone
	default:
		log.Println("notReached colHeader.Tapped")
	}
	c.Refresh()
	if c.OnSortChanged != nil {
		c.OnSortChanged(c.Sort)
	}
}

func (c *colHeader) TappedSecondary(e *fyne.PointEvent) {
	if c.OnTappedSecondary != nil {
		c.OnTappedSecondary(e)
	}
}

func (c *colHeader) Refresh() {
	if c.Sort == SortDescending {
		c.sortIcon.Resource = theme.MenuDropDownIcon()
	} else {
		c.sortIcon.Resource = theme.MenuDropUpIcon()
	}

	if c.Sort > 0 && c.sortIcon.Hidden {
		c.sortIcon.Show()
		c.container.Add(c.sortIconNegSpacer)
	} else if (c.Sort == SortNone || *c.sortDisabled) && !c.sortIcon.Hidden {
		c.sortIcon.Hide()
		c.container.Remove(c.sortIconNegSpacer)
	}

	c.BaseWidget.Refresh()
}

func (c *colHeader) CreateRenderer() fyne.WidgetRenderer {
	if c.container == nil {
		c.container = container.New(&layouts.HboxCustomPadding{DisableThemePad: true, ExtraPad: -8})
		if c.columnCfg.Alignment != fyne.TextAlignLeading {
			c.container.Add(layout.NewSpacer())
		}
		c.container.Add(c.label)
		c.container.Add(c.sortIcon)
		c.container.Add(c.sortIconNegSpacer)
		if c.columnCfg.Alignment == fyne.TextAlignCenter {
			c.container.Add(layout.NewSpacer())
		}
	}
	return widget.NewSimpleRenderer(c.container)
}
