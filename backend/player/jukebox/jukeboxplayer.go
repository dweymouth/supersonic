package jukebox

import (
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/player"
)

const (
	stopped = 0
	playing = 1
	paused  = 2
)

type JukeboxPlayer struct {
	provider mediaprovider.JukeboxProvider

	state     int // stopped, playing, paused
	volume    int
	seeking   bool
	numTracks int

	curTrack          int
	curTrackDuration  float64
	startTrackTime    float64
	startedAtUnixSecs float64
}

func (j *JukeboxPlayer) SetVolume(vol int) error {
	go func() {
		if err := j.provider.JukeboxSetVolume(vol); err == nil {
			j.volume = vol
		}
	}()
	return nil
}

func (j *JukeboxPlayer) GetVolume() int {
	return j.volume
}

func (j *JukeboxPlayer) PlayTrackAt(idx int) error {
	go func() {
		if err := j.provider.JukeboxSeek(idx, 0); err == nil {
			j.curTrack = idx
			j.Continue()
		}
	}()
	return nil
}

func (j *JukeboxPlayer) Continue() error {
	if j.state == playing {
		return nil
	}
	go func() {
		if err := j.provider.JukeboxStart(); err != nil {
			return
		}
		j.state = playing
	}()
	return nil
}

func (j *JukeboxPlayer) Pause() error {
	if j.state != playing {
		return nil
	}
	go func() {
		if err := j.provider.JukeboxStop(); err != nil {
			return
		}
		j.state = paused
	}()
	return nil
}

func (j *JukeboxPlayer) Stop() error {
	if j.state == stopped {
		return nil
	}
	go func() {
		if err := j.provider.JukeboxStop(); err != nil {
			return
		}
		j.state = stopped
	}()
	return nil
}

func (j *JukeboxPlayer) SeekPrevious() error {
	track := j.curTrack
	if track > 0 {
		track = j.curTrack - 1
	}
	return j.PlayTrackAt(track)
}

func (j *JukeboxPlayer) SeekNext() error {
	track := j.curTrack
	if track >= j.numTracks {
		return nil
	}
	return j.PlayTrackAt(track + 1)
}

func (j *JukeboxPlayer) SeekSeconds(secs float64) error {
	j.seeking = true
	go func() {
		j.provider.JukeboxSeek(j.curTrack, int(secs))
		j.seeking = false
	}()
	return nil
}

func (j *JukeboxPlayer) IsSeeking() bool {
	return j.seeking
}

func (j *JukeboxPlayer) GetStatus() player.Status {
	state := player.Stopped
	if j.state == playing {
		state = player.Playing
	} else if j.state == paused {
		state = player.Paused
	}

	// TODO - the rest

	return player.Status{
		State: state,
	}
}
