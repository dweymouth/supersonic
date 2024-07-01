package res

const (
	AppName          = "supersonic"
	DisplayName      = "Supersonic"
	AppVersion       = "0.12.0"
	AppVersionTag    = "v" + AppVersion
	ConfigFile       = "config.toml"
	GithubURL        = "https://github.com/dweymouth/supersonic"
	LatestReleaseURL = GithubURL + "/releases/latest"
	KofiURL          = "https://ko-fi.com/dweymouth"
	Copyright        = "Copyright © 2022–2024 Drew Weymouth and contributors"
)

var (
	WhatsAdded = `
## Added
* Enable drag-and-drop reordering of tracks in the play queue and playlists
* Add command-line options to control playback
* Add option to show album years in grid views
* Include radio station results in Quick Search
* Better stringification of play times longer than 1 hour
* Add fallback logic for populating related tracks and artist top tracks if server returns none
`

	WhatsFixed = `
## Fixed
* Window occasionally misrendered into smaller space on opening for Linux over xwayland
* Add "Play next/later" options to the related tracks list on the Now Playing page
* Change wording of the Add/Edit server form to be less confusing
* Don't crash if server returns nil saved play queue but no Subsonic error
`
)
