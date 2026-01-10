package res

const (
	AppName          = "supersonic"
	DisplayName      = "Supersonic"
	AppVersion       = "0.20.0"
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
* Quickly fade out audio when pausing
* Add collapsed/compact view for page headers
* Add option to prevent screensaver or screen sleep on Now Playing page
* Add lyrics tab to sidebar
* Add additional tracklist columns for Genre, Album Artist, File Type, Date Added
* Show year on Now Playing page
* Option to round image corners throughout app
* Option to used blurred album cover for background of Now Playing page
* Left click on system tray icon raises app window
* Button in Advanced tab of settings window to clear caches
* Seeking Next while playing the last track in queue now cycles around back to the beginning
* Behavior and naming of "Stop after current track" changed to "Pause after current track"`

	WhatsFixed = `
## Fixed
* LrcLib fetcher no longer crashes on tracks with no artist name
* Windows now playing notifications are now silent, and include album cover`
)
