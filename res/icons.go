package res

import (
	"fyne.io/fyne/v2"
	_ "embed"
)

//go:embed super-wave.svg
var SuperWaveIconData []byte

//go:embed super-fly.png
var SuperFlyIconData []byte

//go:embed super-coffe.jpg
var SuperCoffeIconData []byte

//go:embed super-play.jpg
var SuperPlayIconData []byte

func GetAppIcon(name string) fyne.Resource {
	switch name {
	case "Super-wave":
		return fyne.NewStaticResource("super-wave.svg", SuperWaveIconData)
	case "Super-fly":
		return fyne.NewStaticResource("super-fly.png", SuperFlyIconData)
	case "Super-coffee":
		return fyne.NewStaticResource("super-coffe.jpg", SuperCoffeIconData)
	case "Super-play":
		return fyne.NewStaticResource("super-play.jpg", SuperPlayIconData)
	default:
		return ResAppicon256Png
	}
}
