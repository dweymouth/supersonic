package res

const (
	AppName          = "supersonic"
	DisplayName      = "Supersonic"
	AppVersion       = "0.13.0"
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
* Add support for translations, and add Chinese and Italian translations
* Add a peak/RMS meter visualization
* Add track info dialog and context menu item
* Add support for Composer (new tracklist column, and row in Track Info dialog)
* Use artist sortName for sorting artist grid by name, if present
* Add button to sort artist discography by name or year (asc or desc)
* Prevent Windows from sleeping while music playing
* Add a new button below the volume control to show a pop-up play queue
* Add a config file option to disable SSL/TLS validation (useful for self-signed certificates)`

	WhatsFixed = `
## Fixed
* Japanese and possibly other scripts not truncating properly in grid views
* Crash if navigating away from Artist page before cover image loaded
* Regression in not detecting dark/light mode for Linux
* Artist page not loading artist image for servers that don't support artist largeImageURL
* Memory leak when querying certain MPV properties
* Fixed handling of multiple instances of the same track in the play queue
* Improve metadata in Linux .desktop file
* Window occasionally misrendered into smaller space on opening for Linux over xwayland (more reliable fix than last release)`
)
