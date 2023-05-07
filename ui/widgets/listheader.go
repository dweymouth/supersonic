package widgets

import (
	"log"

	"github.com/dweymouth/supersonic/ui/layouts"
	myTheme "github.com/dweymouth/supersonic/ui/theme"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type ListColumn struct {
	Text             string
	AlignTrailing    bool
	CanToggleVisible bool
}

type ListHeader struct {
	widget.BaseWidget

	OnColumnVisibilityChanged   func(int, bool)
	OnColumnVisibilityMenuShown func(*widget.PopUp)

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
	for i, _ := range l.columnVisible {
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
	for _, c := range l.columns {
		t := widget.NewRichTextWithText(c.Text)
		t.Segments[0].(*widget.TextSegment).Style.TextStyle.Bold = true
		al := fyne.TextAlignLeading
		if c.AlignTrailing {
			al = fyne.TextAlignTrailing
		}
		t.Segments[0].(*widget.TextSegment).Style.Alignment = al
		l.columnsContainer.Add(t)
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
