package res

const (
	AppName          = "supersonic"
	DisplayName      = "Supersonic"
	AppVersion       = "0.13.1"
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
* New translations: French, Polish, Portuguese (Brazillian), Romanian, Spanish
* Add tool tips to buttons and truncated labels
* Remember the "skip duplicate tracks" setting when adding tracks to playlists
* Switch sorting on All Tracks page to recently added`

	WhatsFixed = `
## Fixed
* Custom themes didn't work on Windows
* Pop up play queue crashing on Windows
* Fixed "jumpy" scrolling on MacOS
* Window sometimes misdrawing on startup for Linux (even better fix than last time)
* Crash when rendering certain bitmap fonts
* Fixed some more memory leaks from communicating with MPV
* Missing space in error dialog text for sharing
* Play queue sometimes not saving on exit`
)
