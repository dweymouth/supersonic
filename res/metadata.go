package res

const (
	AppName          = "supersonic"
	DisplayName      = "Supersonic"
	AppVersion       = "0.15.0"
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
* Add ability to cast playback to DLNA devices
* Add ability to search tracks within playlist
* Add context menu actions to items in "Search Everywhere" dialog
* Make server requests timeout configurable and increase default timeout
* New overlay animation when showing full size covers`

	WhatsFixed = `
## Fixed
* Wrong track scrobbled when skipping next
* Regression: searching the "All Tracks" page skips some results`
)
