package res

const (
	AppName          = "supersonic"
	DisplayName      = "Supersonic"
	AppVersion       = "0.10.1"
	AppVersionTag    = "v" + AppVersion
	ConfigFile       = "config.toml"
	GithubURL        = "https://github.com/dweymouth/supersonic"
	LatestReleaseURL = GithubURL + "/releases/latest"
	KofiURL          = "https://ko-fi.com/dweymouth"
	Copyright        = "Copyright © 2022–2024 Drew Weymouth and contributors"
)

var (
	WhatsAdded = ``

	WhatsFixed = `
## Fixed
* Crashing when trying to add a new server
* Lyrics don't refresh when playing next song
* What's New dialog wasn't showing when launching updated version
* Crash on exit when saving play queue to server and no track playing
`
)
