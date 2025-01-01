package res

const (
	AppName          = "supersonic"
	DisplayName      = "Supersonic"
	AppVersion       = "0.13.2"
	AppVersionTag    = "v" + AppVersion
	ConfigFile       = "config.toml"
	GithubURL        = "https://github.com/dweymouth/supersonic"
	LatestReleaseURL = GithubURL + "/releases/latest"
	KofiURL          = "https://ko-fi.com/dweymouth"
	Copyright        = "Copyright © 2022–2025 Drew Weymouth and contributors"
)

var (
	WhatsAdded = `
## Added
* New translations: German, Japanese
* Setting in config dialog to choose app language
* Add config file setting to change how many tracks queued by "Play random"
* Clicking on playlist cover from playlist page shows cover in pop-up (like album page)`

	WhatsFixed = `
## Fixed
* Fix regression in scrolling performance of album grid introduced in 0.12.0
* Scroll to currently playing track when loading Now Playing page
* Track with new sample rate failing to begin playback on Windows
* Removing track from a playlist would remove all copies of it
* Use custom HTTP User-Agent to avoid being blocked by some WAF servers
* Increase base scrolling speed, add keybindings for PageUp/PageDown
* Some performance improvements from updating to Fyne 2.5.3
* Ensure clicking outside of tracklists can always unselect selection
* A few translation updates
* Wrong tooltip on repeat control`
)
