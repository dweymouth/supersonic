package dialogs

import (
	"net/url"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/supersonic/res"
	"github.com/dweymouth/supersonic/ui/layouts"
)

type WhatsNewDialog struct {
	widget.BaseWidget
}

func NewWhatsNewDialog() *WhatsNewDialog {
	w := &WhatsNewDialog{}
	w.ExtendBaseWidget(w)
	return w
}

func (w *WhatsNewDialog) CreateRenderer() fyne.WidgetRenderer {
	objects := container.NewVBox()
	if res.WhatsAdded != "" {
		added := widget.NewRichTextFromMarkdown(res.WhatsAdded)
		added.Wrapping = fyne.TextWrapWord
		objects.Add(added)
	}
	if res.WhatsFixed != "" {
		fixed := widget.NewRichTextFromMarkdown(res.WhatsFixed)
		fixed.Wrapping = fyne.TextWrapWord
		objects.Add(fixed)
	}

	ghUrl, _ := url.Parse(res.GithubURL)
	kofiUrl, _ := url.Parse(res.KofiURL)
	githubKofi := container.NewCenter(
		container.New(&layouts.HboxCustomPadding{DisableThemePad: true, ExtraPad: -10},
			widget.NewHyperlink("Github page", ghUrl),
			widget.NewLabel("·"),
			widget.NewHyperlink("Support the project", kofiUrl)))
	objects.Add(githubKofi)

	c := container.NewVScroll(objects)
	return widget.NewSimpleRenderer(c)
}

func (w *WhatsNewDialog) MinSize() fyne.Size {
	return fyne.NewSize(400, 250)
}
