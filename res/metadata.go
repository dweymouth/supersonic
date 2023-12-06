package res

const (
	AppName          = "supersonic"
	DisplayName      = "Supersonic"
	AppVersion       = "0.8.1"
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
* Artist Radio on Jellyfin not generating a fresh mix if clicked a second time
* On Jellyfin, a long artist biography could overflow the page header
* Systray icon missing on Linux since 0.7.0
`
)
