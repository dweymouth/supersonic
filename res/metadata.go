package res

const (
	AppName          = "supersonic"
	DisplayName      = "Supersonic"
	AppVersion       = "0.15.2"
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
* Improvements across a few translations
* BPM column added to tracklist
* Animate button overlays when mouse in/out of album grid
* Album grid play button now takes color from theme`

	WhatsFixed = `
## Fixed
* Crashing when loading locally saved play queue with files deleted from server
* Occasionally crashing when removing tracks from quue via the pop-up queue
* Album info dialog wouldn't scroll long description text
* Regression: Peak meter was broken since 0.15.0
* Playlist descriptions and track comments with newlines could overflow on top of other UI elements
* Remember scroll position when navigating the history back to an album or playlist page`
)
