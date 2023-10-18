package backend

/**
* This file handles implementation of MacOS native controls via the native 'MediaPlayer' framework
**/

// #cgo CFLAGS: -x objective-c
// #cgo LDFLAGS: -framework Cocoa -framework MediaPlayer
// #include "mpmediabridge.h"
import (
	"C"
)

import (
	"fmt"
	"unsafe"

	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/player"
)

type MPMediaHandler struct {
	player          *player.Player
	playbackManager *PlaybackManager
	ArtURLLookup    func(trackID string) (string, error)
}

func NewMPMediaHandler(player *player.Player, playbackManager *PlaybackManager) *MPMediaHandler {
	mp := &MPMediaHandler{
		player:          player,
		playbackManager: playbackManager,
	}

	mp.playbackManager.OnSongChange(func(track, _ *mediaprovider.Track) {
		if mp.ArtURLLookup != nil {
			mp.ArtURLLookup(track.CoverArtID)
		}

		cTitle := C.CString(track.Name)
		defer C.free(unsafe.Pointer(cTitle))

		artist := ""
		if len(track.ArtistNames) > 0 {
			artist = track.ArtistNames[0]
		}

		cArtist := C.CString(artist)
		defer C.free(unsafe.Pointer(cArtist))

		// TODO: pass in local URL for image (can be loaded via NSImage)
		C.setNowPlayingInfo(cTitle, cArtist)
	})

	mp.player.OnStopped(func() {
		// TODO: media stopped playing
		fmt.Println("Stopped")
	})

	mp.player.OnPlaying(func() {
		// TODO: media playing
		fmt.Println("Playing")
	})

	mp.player.OnPaused(func() {
		// TODO: media paused
		fmt.Println("Paused")
	})

	return mp
}
