package res

const (
	AppName          = "supersonic"
	DisplayName      = "Supersonic"
	AppVersion       = "0.18.0"
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
* Make play button on Albums page configurable to shuffle or play in order
* Add sample rate, bit depth, and channel count to track info dialog
* Add waveform seekbar
* Add CLI commands to start minimized, and to show/raise the app window
* Add CLI commands to search and play albums, tracks, and playlists
* Add player control buttons to Windows taskbar thumbnail
* Add ability to stop playback after current track finishes`

	WhatsFixed = ``
)
