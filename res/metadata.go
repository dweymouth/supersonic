package res

const (
	AppName          = "supersonic"
	DisplayName      = "Supersonic"
	AppVersion       = "0.14.0"
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
* New translations: Dutch
* Media Key / SMTC support for Windows
* Add an Autoplay mode to continually add related songs when nearing the end of queue
* Shuffle button added to favorite songs view
* Added toast notifications for certain UI actions
* Add album shuffle modes to Albums and Genre pages
* Caching added for LrcLib lyrics, custom LrcLib URL supported in config file
* Use notify-send for notifications on Linux if available
* Add setting to disable automatic scaling adjustment for detected DPI
* New Advanced settings tab exposing more config file settings to the UI
* Favorite/menu icon buttons added to grid cards when hovered
* Show full dates from OpenSubsonic servers when available
* Log to a file instead of stdout on Windows`

	WhatsFixed = `
## Fixed
* Persist repeat setting across restarts
* Removing track from playlist when sorted removed wrong track
* Fullscreen artist images not loading in full resolution
* Fix volume slider scrolling too quickly with some mice
* Loading dots not stopping if canceling a search before completion
* Add list/grid debouncing to improve scrolling smoothness when scrolling very quickly
* Image caching broken on some servers due to track IDs not being legal filenames
* Migrate to Fyne 2.6 for improved stability and performance
* Sometimes crashing on exit`
)
