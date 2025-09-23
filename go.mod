module github.com/dweymouth/supersonic

go 1.23.0

require (
	fyne.io/fyne/v2 v2.6.1
	github.com/20after4/configdir v0.1.1
	github.com/Microsoft/go-winio v0.6.2
	github.com/cenkalti/dominantcolor v1.0.3
	github.com/deluan/sanitize v0.0.0-20230310221930-6e18967d9fc1
	github.com/dweymouth/fyne-advanced-list v0.0.0-20250211191927-58ea85eec72c
	github.com/dweymouth/fyne-tooltip v0.3.0
	github.com/dweymouth/go-jellyfin v0.0.0-20250914000657-c172d6f678bb
	github.com/go-audio/audio v1.0.0
	github.com/go-audio/wav v1.1.0
	github.com/godbus/dbus/v5 v5.1.0
	github.com/google/uuid v1.3.0
	github.com/hashicorp/go-retryablehttp v0.7.7
	github.com/pelletier/go-toml/v2 v2.0.8
	github.com/quarckster/go-mpris-server v1.0.3
	github.com/supersonic-app/fyne-lyrics v0.0.0-20250614151306-b1880a70a410
	github.com/supersonic-app/go-mpv v0.1.1-0.20250822102843-7a8cde5f5449
	github.com/supersonic-app/go-subsonic v0.0.0-20250913173646-cf4fceb19b43
	github.com/supersonic-app/go-upnpcast v0.0.0-20250330154256-b957204209a5
	github.com/zalando/go-keyring v0.2.6
	golang.org/x/net v0.38.0
	golang.org/x/sys v0.31.0
	golang.org/x/term v0.30.0
	golang.org/x/text v0.23.0
)

require (
	al.essio.dev/pkg/shellescape v1.5.1 // indirect
	fyne.io/systray v1.11.0 // indirect
	github.com/BurntSushi/toml v1.4.0 // indirect
	github.com/charlievieth/strcase v0.0.5 // indirect
	github.com/danieljoos/wincred v1.2.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/fogleman/gg v1.3.0 // indirect
	github.com/fredbi/uri v1.1.0 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/fyne-io/gl-js v0.1.0 // indirect
	github.com/fyne-io/glfw-js v0.2.0 // indirect
	github.com/fyne-io/image v0.1.1 // indirect
	github.com/fyne-io/oksvg v0.1.0 // indirect
	github.com/go-audio/riff v1.0.0 // indirect
	github.com/go-gl/gl v0.0.0-20231021071112-07e5d0ea2e71 // indirect
	github.com/go-gl/glfw/v3.3/glfw v0.0.0-20240506104042-037f3cc74f2a // indirect
	github.com/go-text/render v0.2.0 // indirect
	github.com/go-text/typesetting v0.2.1 // indirect
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0 // indirect
	github.com/h2non/filetype v1.1.3 // indirect
	github.com/hack-pad/go-indexeddb v0.3.2 // indirect
	github.com/hack-pad/safejs v0.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/jeandeaual/go-locale v0.0.0-20241217141322-fcc2cadd6f08 // indirect
	github.com/jsummers/gobmp v0.0.0-20230614200233-a9de23ed2e25 // indirect
	github.com/koron/go-ssdp v0.0.5 // indirect
	github.com/nfnt/resize v0.0.0-20180221191011-83c6a9932646 // indirect
	github.com/nicksnyder/go-i18n/v2 v2.5.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rymdport/portal v0.4.1 // indirect
	github.com/srwiley/oksvg v0.0.0-20221011165216-be6e8873101c // indirect
	github.com/srwiley/rasterx v0.0.0-20220730225603-2ab79fcdd4ef // indirect
	github.com/stretchr/testify v1.10.0 // indirect
	github.com/yuin/goldmark v1.7.8 // indirect
	golang.org/x/image v0.24.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace fyne.io/fyne/v2 v2.6.1 => github.com/dweymouth/fyne/v2 v2.3.0-rc1.0.20250712002006-5064d705dac4

replace github.com/go-audio/wav v1.1.0 => github.com/dweymouth/go-wav v0.0.0-20250719173115-e60429a83eb0

// fixes https://github.com/dweymouth/supersonic/issues/412 until the fix lands upstream
replace github.com/go-gl/glfw/v3.3/glfw v0.0.0-20240506104042-037f3cc74f2a => github.com/supersonic-app/go-glfw/v3.3/glfw v0.0.0-20250906235349-c09e5a2f6b75
