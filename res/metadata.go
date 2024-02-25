package res

const (
	AppName          = "supersonic"
	DisplayName      = "Supersonic"
	AppVersion       = "0.9.1"
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
* Show visual loading cue when searching and no results yet
* Arrow key scrolling and focus keyboard traversal to grid and list views
* Ctrl+Q keyboard shortcut to quit app on Windows and Linux
`

	WhatsFixed = `
## Fixed
* A few MPRIS bugs (thanks @adamantike!)
* Show modal dialog when connecting to server to block UI interaction which could cause crash
* High CPU use when certain dialogs were shown
* Make tracklist widget more efficient
* Lower CPU use when text entry fields are focused
`
)
