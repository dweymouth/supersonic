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
	client        *subsonic.Client
	player        *player.Player

	playQueue        []*subsonic.Child
	nowPlayingIdx    int64
	onSongChange     []func(*subsonic.Child)
	onPlayTimeUpdate []func(float64, float64)
}

func NewPlaybackManager(ctx context.Context, cli *subsonic.Client, p *player.Player) *PlaybackManager {
	pm := &PlaybackManager{
		ctx:    ctx,
		client: cli,
		player: p,
	}
	p.OnTrackChange(func(tracknum int64) {
		pm.nowPlayingIdx = tracknum
		for _, cb := range pm.onSongChange {
			cb(pm.NowPlaying())
		}
		if pm.pollingTick != nil {
			pm.pollingTick.Reset(pm.getPollSpeed())
		}
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
	// TODO: somehow ran into an index out of range crash here
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
	album, err := p.client.GetAlbum(albumID)
	if err != nil {
		return err
	}
	if !appendToQueue {
		p.player.Stop()
		p.playQueue = nil
	}
	for _, song := range album.Song {
		url, err := p.client.GetStreamURL(song.ID, map[string]string{})
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

// depending on the length of the current track, we need to poll
// faster or less fast to make track position scroll bar look smooth
func (p *PlaybackManager) getPollSpeed() time.Duration {
	t := p.player.GetStatus().Duration
	if t < 30 {
		return 100 * time.Millisecond
	} else if t < 90 {
		return 150 * time.Millisecond
	} else if t < 120 {
		return 250 * time.Millisecond
	} else {
		return 333 * time.Millisecond
	}
}

func (p *PlaybackManager) startPollTimePos() {
	ctx, cancel := context.WithCancel(p.ctx)
	p.cancelPollPos = cancel
	p.pollingTick = time.NewTicker(p.getPollSpeed())
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
}
