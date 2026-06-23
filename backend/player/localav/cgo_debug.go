//go:build debug && !mpv

package localav

/*
#cgo CFLAGS: -DSUPERSONIC_AUDIO_DEBUG=1
*/
import "C"
