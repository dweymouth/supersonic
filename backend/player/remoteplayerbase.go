package player

import (
	"sync/atomic"
	"time"
)

type RemotePlayerBase struct {
	BasePlayerCallbackImpl

	// timer to handle next track change
	timerActive atomic.Bool
	timer       *time.Timer
	resetChan   chan (time.Duration)

	onTrackChange func()
}

func (r *RemotePlayerBase) InitRemotePlayerBase(handleOnTrackChange func()) {
	r.resetChan = make(chan time.Duration)
	r.onTrackChange = handleOnTrackChange
}

func (r *RemotePlayerBase) SetTrackChangeTimer(dur time.Duration) {
	if r.timerActive.Swap(true) {
		// was active
		r.resetChan <- dur
		return
	}
	if dur == 0 {
		r.timerActive.Store(false)
		return
	}

	r.timer = time.NewTimer(dur)
	go func() {
		for {
			select {
			case dur := <-r.resetChan:
				if dur == 0 {
					r.timerActive.Store(false)
					if !r.timer.Stop() {
						select {
						case <-r.timer.C:
						default:
						}
					}
					r.timer = nil
					return
				}
				// reset the timer
				if !r.timer.Stop() {
					select {
					case <-r.timer.C:
					default:
					}
				}
				r.timer.Reset(dur)
			case <-r.timer.C:
				r.timerActive.Store(false)
				r.timer = nil
				r.onTrackChange()
				return
			}
		}
	}()
}
