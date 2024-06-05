package res

const (
	AppName          = "supersonic"
	DisplayName      = "Supersonic"
	AppVersion       = "0.11.0"
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
* Automatically load fonts from the OS as needed to display CJK and other scripts
* Synced lyrics support and Jellyfin lyrics support
* LrcLib.net as a backup lyric source (disable in config file if desired)
* Improve UX of Add to Playlist dialog
* Show track thumbnails in tracklist views
* Support for internet radio stations for Subsonic servers
* New "option button" to right of current track title to bring up action menu
* Ctrl+{backspace/delete} to remove words in text inputs
* New portable mode option
* Improves performance and behavior of Random albums sort with upcoming Navidrome releases
* Dynamic gradient background on Now Playing page
`

	WhatsFixed = `
## Fixed
* Last track occasionally missing in album view depending on window size
* Album filter button disappearing when restoring pages from history
* Artist radio button sometimes plays radio for wrong artist
* Clicking Home button doesn't automatically refresh page
`
)
