package res

const (
	AppName          = "supersonic"
	DisplayName      = "Supersonic"
	AppVersion       = "0.21.1"
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
* Turkish translation`

	WhatsFixed = `
## Fixed
* Regression: library not loading for Funkwhale servers
* Fix crash when playing track with index shorter than previous queue length while shuffling
* Fix intermittent crash when creating share URLs
* Close to dock behavior for MacOS
* Regression: Sorting artist discography not working
* Shuffle button on albums and playlists not working after introduction of queue shuffle mode feature`
)
