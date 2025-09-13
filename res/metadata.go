package res

const (
	AppName          = "supersonic"
	DisplayName      = "Supersonic"
	AppVersion       = "0.18.1"
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
* Save and use the most recently selected music library when re-launching app
* Add Artists to the supported list of startup pages`

	WhatsFixed = `
## Fixed
* Occasionally crashing on Linux when monitors go idle
* AppImage: failing to start due to "No GLXFBConfigs returned
* AppImage: failing to start on openSUSE-Tumbleweed and CachyOS
* Rework format for now playing status line
* Silenty failing to add large number of tracks to playlist (OpenSubsonic servers)
* Fail to start on Windows ARM64 or older x64 CPUs
* Search results limited to 20 items
* Transcoding not working with Jellyfin
* Crashing on "Shuffle albums" if fewer than 20 albums available`
)
