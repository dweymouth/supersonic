package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	myTheme "github.com/dweymouth/supersonic/ui/theme"
)

// AdaptiveHeader wraps content with a themed background that can be toggled
// between opaque and transparent. Used for page headers that need to
// adapt when background blur/gradient is enabled.
type AdaptiveHeader struct {
	widget.BaseWidget

	Content       fyne.CanvasObject
	ColorName     fyne.ThemeColorName

	bgRect        *myTheme.ThemedRectangle
	container     *fyne.Container
	isTransparent bool
}

// NewAdaptiveHeader creates a new header with the given content and color.
func NewAdaptiveHeader(content fyne.CanvasObject, colorName fyne.ThemeColorName) *AdaptiveHeader {
	a := &AdaptiveHeader{
		Content:   content,
		ColorName: colorName,
	}
	a.ExtendBaseWidget(a)
	return a
}

// SetTransparent toggles the background transparency.
// When transparent, the themed background is hidden.
func (a *AdaptiveHeader) SetTransparent(transparent bool) {
	if a.isTransparent != transparent {
		a.isTransparent = transparent
		if a.bgRect != nil {
			if transparent {
				a.bgRect.Hide()
			} else {
				a.bgRect.Show()
			}
		}
	}
}

// IsTransparent returns the current transparency state.
func (a *AdaptiveHeader) IsTransparent() bool {
	return a.isTransparent
}

// CreateRenderer creates the renderer for this widget.
func (a *AdaptiveHeader) CreateRenderer() fyne.WidgetRenderer {
	if a.container == nil {
		a.bgRect = myTheme.NewThemedRectangle(a.ColorName)
		// Note: Corner radius can be set here if needed via theme or passed parameter

		if a.isTransparent {
			a.bgRect.Hide()
		}

		padded := container.New(&layout.CustomPaddedLayout{LeftPadding: 10, RightPadding: 10, TopPadding: 10, BottomPadding: 10}, a.Content)
		a.container = container.NewStack(a.bgRect, padded)
	}
	return widget.NewSimpleRenderer(a.container)
}
