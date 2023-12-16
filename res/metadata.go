package res

const (
	AppName          = "supersonic"
	DisplayName      = "Supersonic"
	AppVersion       = "0.8.2"
	AppVersionTag    = "v" + AppVersion
	ConfigFile       = "config.toml"
	GithubURL        = "https://github.com/dweymouth/supersonic"
	LatestReleaseURL = GithubURL + "/releases/latest"
	KofiURL          = "https://ko-fi.com/dweymouth"
)

var (
	WhatsAdded = ``

	WhatsFixed = `
## Fixed
* Occasional crash when showing album info dialog, especially on Jellyfin
* Unable to connect to Airsonic servers not supporting latest Subsonic API
* Long album titles overflow the bounds of info dialog
* ReplayGain "prevent clipping" setting was reversed
`
)
