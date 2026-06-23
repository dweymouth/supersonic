//go:build linux && !mpv

package localav

/*
#cgo pkg-config: libavformat libavcodec libavfilter libavutil libswresample wavpack
#cgo LDFLAGS: -lpthread -ldl -lm
*/
import "C"
