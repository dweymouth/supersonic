package res

const (
	AppName          = "supersonic"
	DisplayName      = "Supersonic"
	AppVersion       = "0.15.1"
	AppVersionTag    = "v" + AppVersion
	ConfigFile       = "config.toml"
	GithubURL        = "https://github.com/dweymouth/supersonic"
	LatestReleaseURL = GithubURL + "/releases/latest"
	KofiURL          = "https://ko-fi.com/dweymouth"
	Copyright        = "Copyright © 2022–2025 Drew Weymouth and contributors"
)

var (
	WhatsAdded = ``

	WhatsFixed = `
## Fixed
* Fix crashing on playback on Windows
* Fix crashing when clicking on album filters button`
)
