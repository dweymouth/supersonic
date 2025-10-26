package res

const (
	AppName          = "supersonic"
	DisplayName      = "Supersonic"
	AppVersion       = "0.19.0"
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
* Toggleable side bar play queue
* New setting to skip one-star tracks or tracks with specific keyword when shuffling
* New Last Played column for tracklists
* Create Playlist button added to the playlists page
* Save play queue on all queue changes rather than on shutdown
* Left and Right keybindings for seeking back/forward 10 sec`

	WhatsFixed = `
## Fixed
* Peak Meter visualization crashing with custom themes
* Adjust spacing for rating/favorite widget on Now Playing page
* Occasional crashing on lyrics viewer
* Jellyfin M3U playlists not showing up
* Occasional hangs when loading artist pages
* Improved LrcLib lyrics fetching with unknown album or artist name`
)
