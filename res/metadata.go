package res

const (
	AppName          = "supersonic"
	DisplayName      = "Supersonic"
	AppVersion       = "0.20.1"
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
* Improved translations for Chinese, Japanese, Italian
* Setting to disable fade out on pause
* Ignore SSL validation setting migrated to per-server option`

	WhatsFixed = `
## Fixed
* Some uPnP devices not recognized
* Jellyfin synced lyrics not shown as synced if first line starts at 00:00.00
* Regression in peak meter animation smoothness
* Some optimizations of tracklist rendering
* Autoplay tracks enqueuing too early
* Fix equalizer filter string for ffmpeg
* Unnecessarily large minimum window width for pages with tracklist
* Use sort tags for sorting artist discography by album name (OpenSubsonic only)
* For Jellyfin - Add to Playlist dialog not showing playlists when logged in with username in different case
* Fix Now Playing background dissolve animation when next track has same cover art as previous`
)
