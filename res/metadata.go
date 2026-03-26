package res

const (
	AppName          = "supersonic"
	DisplayName      = "Supersonic"
	AppVersion       = "0.21.0"
	AppVersionTag    = "v" + AppVersion
	ConfigFile       = "config.toml"
	GithubURL        = "https://github.com/dweymouth/supersonic"
	LatestReleaseURL = GithubURL + "/releases/latest"
	KofiURL          = "https://ko-fi.com/dweymouth"
	Copyright        = "Copyright © 2022–2026 Drew Weymouth and contributors"
)

var (
	WhatsAdded = `
## Added
* Shuffle play queue
* All Tracks tab on Artist page
* Equalizer improvements: 10-band EQ option, AutoEQ headphone profiles, preset management
* Song Radio context menu item on play queue
* Display radio stream ICY metadata
* -rate-current CLI option to rate the currently playing track
* -reload-theme CLI flag to re-apply the current theme
* Updated Polish and Chinese translations`

	WhatsFixed = `
## Fixed
* Crash in playback engine during time position update
* Clearing cache did not clear in-memory cache
* Normalize server URLs before connecting
* Race condition around context cancel func
* Occasional momentary playback when loading saved play queue
* Fix content not loading on startup until window focused`
)
