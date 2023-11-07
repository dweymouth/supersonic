package res

import (
	"fmt"
	"runtime"
)

const (
	AppName          = "supersonic"
	DisplayName      = "Supersonic"
	AppVersion       = "0.6.0"
	AppVersionTag    = "v" + AppVersion
	ConfigFile       = "config.toml"
	GithubURL        = "https://github.com/dweymouth/supersonic"
	LatestReleaseURL = GithubURL + "/releases/latest"
	KofiURL          = "https://ko-fi.com/dweymouth"
)

func shortcutKey() string {
	if runtime.GOOS == "darwin" {
		return "Cmd"
	}
	return "Ctrl"
}

var (
	WhatsAdded = fmt.Sprintf(`
### Added
* New "Quick search" feature to search entire library from anywhere (%s+G shortcut)
* Support for loading WEBP images
* UI Refresh from migrating to Fyne 2.4
* New [custom theme](https://github.com/dweymouth/supersonic/wiki/Custom-Themes) color keys: Hyperlink and PageHeader (theme file syntax "0.2")
* Added playback setting to force-disable server transcoding`, shortcutKey())

	WhatsFixed = `
### Fixed
* UI hang when playback slow to begin after clicking on grid view play buttons
* Crash when removing from the currently playing track to end of play queue`
)
