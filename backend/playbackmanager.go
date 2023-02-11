package backend

import (
	"context"
	"log"
	"strconv"
	"supersonic/backend/util"
	"supersonic/player"
	"time"

	"github.com/dweymouth/go-subsonic/subsonic"
)

const (
	ScrobbleThreshold = 0.9
)

type PlaybackManager struct {
	ctx           context.Context
	cancelPollPos context.CancelFunc
	pollingTick   *time.Ticker
	sm            *ServerManager
	player        *player.Player

	playTimeStopwatch util.Stopwatch
	curTrackTime      float64

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
	// TODO: we get some spurious OnTrackChange callbacks from the player,
	// especially when loading/playing a new album,
	// but for now they're pretty much harmless. Investigate later.
	p.OnTrackChange(func(tracknum int64) {
		if tracknum >= int64(len(pm.playQueue)) {
			return
		}
		pm.checkScrobble(pm.playTimeStopwatch.Elapsed())
		pm.playTimeStopwatch.Reset()
		if pm.player.GetStatus().State == player.Playing {
			pm.playTimeStopwatch.Start()
		}
		pm.nowPlayingIdx = tracknum
		pm.curTrackTime = float64(pm.playQueue[pm.nowPlayingIdx].Duration)
		for _, cb := range pm.onSongChange {
			cb(pm.NowPlaying())
		}
		pm.doUpdateTimePos()
	})
	p.OnSeek(func() {
		pm.doUpdateTimePos()
	})
	p.OnStopped(func() {
		pm.playTimeStopwatch.Stop()
		pm.checkScrobble(pm.playTimeStopwatch.Elapsed())
		pm.playTimeStopwatch.Reset()
		pm.stopPollTimePos()
		pm.doUpdateTimePos()
		for _, cb := range pm.onSongChange {
			cb(nil)
		}
	})
	p.OnPaused(func() {
		pm.playTimeStopwatch.Stop()
		pm.stopPollTimePos()
	})
	p.OnPlaying(func() {
		pm.playTimeStopwatch.Start()
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
	return p.LoadTracks(album.Song, appendToQueue)
}

// Loads the specified playlist into the play queue.
func (p *PlaybackManager) LoadPlaylist(playlistID string, appendToQueue bool) error {
	playlist, err := p.sm.Server.GetPlaylist(playlistID)
	if err != nil {
		return err
	}
	return p.LoadTracks(playlist.Entry, appendToQueue)
}

func (p *PlaybackManager) LoadTracks(tracks []*subsonic.Child, appendToQueue bool) error {
	if !appendToQueue {
		p.player.Stop()
		p.nowPlayingIdx = 0
		p.playQueue = nil
	}
	for _, song := range tracks {
		url, err := p.sm.Server.GetStreamURL(song.ID, map[string]string{})
		if err != nil {
			return err
		}
		p.player.AppendFile(url.String())
		p.playQueue = append(p.playQueue, song)
	}
	return nil
}

func (p *PlaybackManager) PlayAlbum(albumID string, firstTrack int) error {
	if err := p.LoadAlbum(albumID, false); err != nil {
		return err
	}
	if firstTrack <= 0 {
		return p.player.PlayFromBeginning()
	}
	return p.player.PlayTrackAt(firstTrack)
}

func (p *PlaybackManager) PlayPlaylist(playlistID string, firstTrack int) error {
	if err := p.LoadPlaylist(playlistID, false); err != nil {
		return err
	}
	if firstTrack <= 0 {
		return p.player.PlayFromBeginning()
	}
	return p.player.PlayTrackAt(firstTrack)
}

func (p *PlaybackManager) PlayFromBeginning() error {
	return p.player.PlayFromBeginning()
}

func (p *PlaybackManager) PlayTrackAt(idx int) error {
	return p.player.PlayTrackAt(idx)
}

func (p *PlaybackManager) GetPlayQueue() []*subsonic.Child {
	pq := make([]*subsonic.Child, len(p.playQueue))
	copy(pq, p.playQueue)
	return pq
}

// trackIdxs must be sorted
func (p *PlaybackManager) RemoveTracksFromQueue(trackIdxs []int) {
	newQueue := make([]*subsonic.Child, 0, len(p.playQueue)-len(trackIdxs))
	rmCount := 0
	rmIdx := 0
	for i, tr := range p.playQueue {
		if rmIdx < len(trackIdxs) && trackIdxs[rmIdx] == i {
			// removing this track
			// TODO: if we are removing the currently playing track,
			// we need to scrobble it if it played for more than the scrobble threshold
			rmIdx++
			if err := p.player.RemoveTrackAt(i - rmCount); err == nil {
				rmCount++
			} else {
				log.Printf("error removing track: %v", err.Error())
				// did not remove this track
				newQueue = append(newQueue, tr)
			}
		} else {
			// not removing this track
			newQueue = append(newQueue, tr)
		}
	}
	p.playQueue = newQueue
	p.nowPlayingIdx = p.player.GetStatus().PlaylistPos
	// fire on song change callbacks in case the playing track was removed
	// TODO: only call this if the playing track actually was removed
	for _, cb := range p.onSongChange {
		cb(p.NowPlaying())
	}
}

func (p *PlaybackManager) checkScrobble(playDur time.Duration) {
	if len(p.playQueue) == 0 || p.nowPlayingIdx < 0 {
		return
	}
	if playDur.Seconds() < 0.1 || p.curTrackTime < 0.1 {
		return // ignore spurious onTrackChange callbacks
	}
	song := p.playQueue[p.nowPlayingIdx]
	if playDur.Seconds()/p.curTrackTime > ScrobbleThreshold {
		log.Printf("Scrobbling %q", song.Title)
		p.sm.Server.Scrobble(song.ID, map[string]string{"time": strconv.FormatInt(time.Now().Unix()*1000, 10)})
	}
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
