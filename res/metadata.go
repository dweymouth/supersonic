package res

const (
	AppName          = "supersonic"
	DisplayName      = "Supersonic"
	AppVersion       = "0.10.0"
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
* Support mouse back and forward buttons for navigation
* Redesigned Now Playing page with large cover art and lyrics
* Add context menu entry to create sharing URLs (Subsonic servers only)
* Add option to load/save play queue to Subsonic servers instead of locally
* Add option to shuffle artist's discography by albums or tracks
* Show original and reissue date for albums if available (OpenSubsonic servers)
* Add "Song radio" option to tracklist context menus
* Add "Play next" option to context menus
* Add sorting options to Artists page
`

	WhatsFixed = `
## Fixed
* Can't reorder track in play queue more than once
* Playlist's comment text doesn't wrap
* Pressing down arrow on tracklist with few tracks could cause some to be hidden
* "Move down" option in play queue context menu was moving to bottom
`
)
