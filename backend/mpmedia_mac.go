//go:build darwin

package backend

/**
* This file handles implementation of MacOS native controls via the native 'MediaPlayer' framework
**/

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework MediaPlayer
#include "mpmediabridge.h"
*/
import (
	"C"
)

import (
	"fmt"
	"log"
	"strings"
	"unsafe"

	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/player"
)

// os_remote_command_callback is called by Objective-C when incoming OS media commands are received.
//
//export os_remote_command_callback
func os_remote_command_callback(command C.Command, value C.double) {
	switch command {
	case C.PLAY:
		mpMediaEventRecipient.OnCommandPlay()
	case C.PAUSE:
		mpMediaEventRecipient.OnCommandPause()
	case C.STOP:
		mpMediaEventRecipient.OnCommandStop()
	case C.TOGGLE:
		mpMediaEventRecipient.OnCommandTogglePlayPause()
	case C.PREVIOUS_TRACK:
		mpMediaEventRecipient.OnCommandPreviousTrack()
	case C.NEXT_TRACK:
		mpMediaEventRecipient.OnCommandNextTrack()
	case C.SEEK:
		mpMediaEventRecipient.OnCommandSeek(float64(value))
	default:
		log.Printf("unknown OS command received: %v", command)
	}
}

// global recipient for Object-C callbacks from command center.
// This is global so that it can be called from 'os_remote_command_callback' to avoid passing Go pointers into C.
var mpMediaEventRecipient *MPMediaHandler

// NewMPMediaHandler creates a new MPMediaHandler instances and sets it as the current recipient
// for incoming system events.
func NewMPMediaHandler(player *player.Player, playbackManager *PlaybackManager) *MPMediaHandler {
	mp := &MPMediaHandler{
		player:          player,
		playbackManager: playbackManager,
	}

	// register remote commands and set callback target
	mpMediaEventRecipient = mp
	C.register_os_remote_commands()

	mp.playbackManager.OnSongChange(func(track, _ *mediaprovider.Track) {
		if track != nil && track.ID != "" {
			var artURL string
			if mp.ArtURLLookup != nil {
				var err error
				if artURL, err = mp.ArtURLLookup(track.CoverArtID); err != nil {
					log.Printf("error fetching art url: %s", err.Error())
				}
			}

			cTitle := C.CString(track.Name)
			defer C.free(unsafe.Pointer(cTitle))

			var artist string
			if len(track.ArtistNames) > 0 {
				artist = strings.Join(track.ArtistNames, ", ")
			}

			cArtist := C.CString(artist)
			defer C.free(unsafe.Pointer(cArtist))

			cArtURL := C.CString(artURL)
			defer C.free(unsafe.Pointer(cArtURL))

			cTrackDuration := C.double(track.Duration)

			C.set_os_now_playing_info(cTitle, cArtist, cArtURL, cTrackDuration)
		}
	})

	mp.player.OnStopped(func() {
		C.set_os_playback_state_stopped()
	})

	mp.player.OnSeek(func() {
		C.update_os_now_playing_info_position(C.double(mp.player.GetStatus().TimePos))
	})

	mp.player.OnPlaying(func() {
		C.set_os_playback_state_playing()
		C.update_os_now_playing_info_position(C.double(mp.player.GetStatus().TimePos))
	})

	mp.player.OnPaused(func() {
		C.set_os_playback_state_paused()
		C.update_os_now_playing_info_position(C.double(mp.player.GetStatus().TimePos))
	})

	return mp
}

/**
* Handle incoming OS commands.
**/

// MPMediaHandler instance received OS command 'pause'
func (mp *MPMediaHandler) OnCommandPause() {
	if mp == nil || mp.player == nil {
		return
	}
	mp.player.Pause()
}

// MPMediaHandler instance received OS command 'play'
func (mp *MPMediaHandler) OnCommandPlay() {
	if mp == nil || mp.player == nil {
		return
	}
	mp.player.Continue()
}

// MPMediaHandler instance received OS command 'stop'
func (mp *MPMediaHandler) OnCommandStop() {
	if mp == nil || mp.player == nil {
		return
	}
	mp.player.Stop()
}

// MPMediaHandler instance received OS command 'toggle'
func (mp *MPMediaHandler) OnCommandTogglePlayPause() {
	if mp == nil || mp.player == nil {
		return
	}
	if mp.player.GetStatus().State == player.Playing {
		mp.OnCommandPause()
	} else {
		mp.OnCommandPlay()
	}
}

// MPMediaHandler instance received OS command 'next track'
func (mp *MPMediaHandler) OnCommandNextTrack() {
	if mp == nil || mp.player == nil {
		return
	}
	mp.player.SeekNext()
}

// MPMediaHandler instance received OS command 'previous track'
func (mp *MPMediaHandler) OnCommandPreviousTrack() {
	if mp == nil || mp.player == nil {
		return
	}
	mp.player.SeekBackOrPrevious()
}

// MPMediaHandler instance received OS command to 'seek'
func (mp *MPMediaHandler) OnCommandSeek(positionSeconds float64) {
	if mp == nil || mp.player == nil {
		return
	}
	mp.player.Seek(fmt.Sprintf("%0.2f", positionSeconds), player.SeekAbsolute)
}
