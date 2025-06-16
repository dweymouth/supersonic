package res

const (
	AppName          = "supersonic"
	DisplayName      = "Supersonic"
	AppVersion       = "0.16.0"
	AppVersionTag    = "v" + AppVersion
	ConfigFile       = "config.toml"
	GithubURL        = "https://github.com/dweymouth/supersonic"
	LatestReleaseURL = GithubURL + "/releases/latest"
	KofiURL          = "https://ko-fi.com/dweymouth"
	Copyright        = "Copyright © 2022–2025 Drew Weymouth and contributors"
)

var (
	WhatsAdded = `
## Added
* Add click-to-seek to lyrics viewer for synced lyrics
* Allow customization of grid card size
* Add setting to request transcoding to a specific format/bitrate from server
* Japanese translation improvements`

	WhatsFixed = `
## Fixed
* New setting to disable OS Now Playing APIs - workaround for Windows SMTC bugginess
* Crashing with custom themes that change the Primary color
* Incorrect tracks removed from playlist if removing from searched view`
)
