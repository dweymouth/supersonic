package res

const (
	AppName          = "supersonic"
	DisplayName      = "Supersonic"
	AppVersion       = "0.8.0"
	AppVersionTag    = "v" + AppVersion
	ConfigFile       = "config.toml"
	GithubURL        = "https://github.com/dweymouth/supersonic"
	LatestReleaseURL = GithubURL + "/releases/latest"
	KofiURL          = "https://ko-fi.com/dweymouth"
)

var (
	WhatsAdded = `
## Added
* Add support for connecting to Jellyfin servers
* Add "Auto" ReplayGain option to auto-choose between Track and Album mode
* Add experimental setting for changing UI scaling`

	WhatsFixed = `
## Fixed
* Crash when repeatedly searching the All Tracks page quickly
* What's New dialog sometimes continuing to re-show on subsequent launches
* Config settings sometimes not being saved due to abnormal exits
* Don't crash with zero-track albums or tracks with no artists
* Slightly improved the time it takes to check server connectivity`
)
