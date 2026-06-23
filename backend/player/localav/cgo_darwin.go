//go:build darwin && !mpv

package localav

/*
#cgo pkg-config: libavformat libavcodec libavfilter libavutil libswresample libswscale wavpack
#cgo LDFLAGS: -framework CoreAudio -framework AudioToolbox -framework CoreFoundation
*/
import "C"
