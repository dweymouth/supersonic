package res

const (
	AppName          = "supersonic"
	DisplayName      = "Supersonic"
	AppVersion       = "0.17.0"
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
* Group releases by Release Type on artist pages
* Allow browsing a specific music library for servers that expose multiple libraries
* New translations: Korean and Russian
* Some updates to other translations`

	WhatsFixed = `
## Fixed
* Light button colors in a light theme turn dark when hovering
* Use full date instead of year to sort artist discography
* Improved rendering of text entry borders when using fractional pixel scaling
* Windows: occasional crashes at startup
* Windows: sporadic crashes when changing tracks due to bugs in SMTC DLL
* Windows: TLS versions above 1.2 not supported by shipped libmpv-2
* Windows: 32-bit (fixed-point) FLAC files cannot be played (libmpv built before ffmpeg support existed)
* Windows: some radio stream URLs wouldn't play`
)
