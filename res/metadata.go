package res

const (
	AppName          = "supersonic"
	DisplayName      = "Supersonic"
	AppVersion       = "0.9.0"
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
* Allow reordering of tracks in the play queue
* Highlight the icon of the current page's navigation button (thanks @natilou!)
* Show release type in album page header (for OpenSubsonic servers)
* Setting to save and reload play queue on exit/startup
* Use most recent playlist as default in "Add to playlist" dialog
* Option to show desktop notifications on track change
* Added icons to context menu items
`

	WhatsFixed = `
## Fixed
* OpenGL startup error on some hardware
`
)
