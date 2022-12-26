package backend

import (
	"context"
	"supersonic/player"
	"time"

	subsonic "github.com/dweymouth/go-subsonic"
)

type PlaybackManager struct {
	ctx           context.Context
	cancelPollPos context.CancelFunc
	pollingTick   *time.Ticker
	sm            *ServerManager
	player        *player.Player

	playQueue        []*subsonic.Child
	nowPlayingIdx    int64
	onSongChange     []func(*subsonic.Child)
	onPlayTimeUpdate []func(float64, float64)
}

func NewPlaybackManager(ctx context.Context, s *ServerManager, p *player.Player) *PlaybackManager {
	pm := &PlaybackManager{
		ctx:    ctx,
		sm:     s,
		player: p,
	}
	p.OnTrackChange(func(tracknum int64) {
		pm.nowPlayingIdx = tracknum
		for _, cb := range pm.onSongChange {
			cb(pm.NowPlaying())
		}
		pm.doUpdateTimePos()
	})
	p.OnSeek(func() {
		pm.doUpdateTimePos()
	})
	p.OnStopped(func() {
		pm.stopPollTimePos()
		for _, cb := range pm.onSongChange {
			cb(nil)
		}
	})
	p.OnPaused(func() {
		pm.stopPollTimePos()
	})
	p.OnPlaying(func() {
		pm.startPollTimePos()
	})

	return pm
}

func (p *PlaybackManager) IsSeeking() bool {
	return p.player.IsSeeking()
}

// Gets the curently playing song, if any.
func (p *PlaybackManager) NowPlaying() *subsonic.Child {
	if len(p.playQueue) == 0 || p.player.GetStatus().State == player.Stopped {
		return nil
	}
	return p.playQueue[p.nowPlayingIdx]
}

// Sets a callback that is notified whenever a new song begins playing.
func (p *PlaybackManager) OnSongChange(cb func(*subsonic.Child)) {
	p.onSongChange = append(p.onSongChange, cb)
}

// Registers a callback that is notified whenever the play time should be updated.
func (p *PlaybackManager) OnPlayTimeUpdate(cb func(float64, float64)) {
	p.onPlayTimeUpdate = append(p.onPlayTimeUpdate, cb)
}

// Loads the specified album into the play queue.
func (p *PlaybackManager) LoadAlbum(albumID string, appendToQueue bool) error {
	album, err := p.sm.Server.GetAlbum(albumID)
	if err != nil {
		return err
	}
	if !appendToQueue {
		p.player.Stop()
		p.nowPlayingIdx = 0
		p.playQueue = nil
	}
	for _, song := range album.Song {
		url, err := p.sm.Server.GetStreamURL(song.ID, map[string]string{})
		if err != nil {
			return err
		}
		p.player.AppendFile(url.String())
		p.playQueue = append(p.playQueue, song)
	}
	return nil
}

func (p *PlaybackManager) PlayAlbum(albumID string) error {
	if err := p.LoadAlbum(albumID, false); err != nil {
		return err
	}
	return p.player.PlayFromBeginning()
}

func (p *PlaybackManager) startPollTimePos() {
	ctx, cancel := context.WithCancel(p.ctx)
	p.cancelPollPos = cancel
	p.pollingTick = time.NewTicker(250 * time.Millisecond)

	// TODO: fix occasional nil pointer dereference on app quit
	go func() {
		for {
			select {
			case <-ctx.Done():
				p.pollingTick.Stop()
				p.pollingTick = nil
				return
			case <-p.pollingTick.C:
				p.doUpdateTimePos()
			}
		}
	}()
}

func (p *PlaybackManager) doUpdateTimePos() {
	s := p.player.GetStatus()
	for _, cb := range p.onPlayTimeUpdate {
		cb(s.TimePos, s.Duration)
	}
}

func (p *PlaybackManager) stopPollTimePos() {
	if p.cancelPollPos != nil {
		p.cancelPollPos()
		p.cancelPollPos = nil
	}
	if p.pollingTick != nil {
		p.pollingTick.Stop()
	}
}
