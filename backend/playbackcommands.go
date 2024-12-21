package backend

import (
	"slices"
	"sync"

	"github.com/dweymouth/supersonic/backend/mediaprovider"
)

type playbackCommandType int

const (
	cmdStop playbackCommandType = iota
	cmdContinue
	cmdPause
	cmdPlayTrackAt  // arg: int
	cmdSeekSeconds  // arg: float64
	cmdSeekFwdBackN // arg: int
	cmdVolume       // arg: int
	cmdLoopMode     // arg: LoopMode
	cmdStopAndClearPlayQueue
	cmdUpdatePlayQueue       // arg: []mediaprovider.MediaItem
	cmdRemoveTracksFromQueue // arg: []int
	// arg: []mediaprovider.MediaItem
	// arg2: InsertMode
	// arg3: bool (shuffle)
	cmdLoadItems
	cmdLoadRadioStation // arg: *mediaprovider.RadioStation, arg2: InsertQueueMode

	cmdForceRestartPlayback
)

type playbackCommand struct {
	Type playbackCommandType
	Arg  any
	Arg2 any
	Arg3 any
}

// playbackCommandQueue is a queue to accumulate player commands from the UI
// commands are processed by the playback engine as fast as they can, but if
// more commands arrive before the player can respond to them, they will queue up
// and some commands may coalesce together (e.g. multiple volume commands into just one)
type playbackCommandQueue struct {
	mutex        sync.Mutex
	queue        []playbackCommand
	cmdAvailable *sync.Cond
	nextChan     chan playbackCommand
}

func NewCommandQueue() *playbackCommandQueue {
	c := &playbackCommandQueue{}
	c.nextChan = make(chan playbackCommand)
	c.cmdAvailable = sync.NewCond(&c.mutex)
	go c.chanWriter()
	return c
}

func (c *playbackCommandQueue) C() <-chan playbackCommand {
	return c.nextChan
}

func (c *playbackCommandQueue) Stop() {
	c.filterCommandsAndAdd([]playbackCommandType{cmdContinue, cmdPause, cmdStop},
		playbackCommand{Type: cmdStop})
}

func (c *playbackCommandQueue) Continue() {
	c.filterCommandsAndAdd([]playbackCommandType{cmdContinue, cmdPause, cmdStop},
		playbackCommand{Type: cmdContinue})
}

func (c *playbackCommandQueue) Pause() {
	c.filterCommandsAndAdd([]playbackCommandType{cmdContinue, cmdPause, cmdStop},
		playbackCommand{Type: cmdPause})
}

func (c *playbackCommandQueue) PlayTrackAt(idx int) {
	c.filterCommandsAndAdd([]playbackCommandType{cmdContinue, cmdPause, cmdStop, cmdPlayTrackAt},
		playbackCommand{Type: cmdPlayTrackAt, Arg: idx})
}

func (c *playbackCommandQueue) StopAndClearPlayQueue() {
	c.filterCommandsAndAdd([]playbackCommandType{cmdContinue, cmdPause, cmdStop, cmdStopAndClearPlayQueue},
		playbackCommand{Type: cmdStopAndClearPlayQueue})
}

func (c *playbackCommandQueue) SetVolume(vol int) {
	c.filterCommandsAndAdd([]playbackCommandType{cmdVolume},
		playbackCommand{Type: cmdVolume, Arg: vol})
}

func (c *playbackCommandQueue) SetLoopMode(mode LoopMode) {
	c.filterCommandsAndAdd([]playbackCommandType{cmdLoopMode},
		playbackCommand{Type: cmdLoopMode, Arg: mode})
}

func (c *playbackCommandQueue) SeekSeconds(s float64) {
	c.filterCommandsAndAdd([]playbackCommandType{cmdSeekSeconds},
		playbackCommand{Type: cmdSeekSeconds, Arg: s})
}

func (c *playbackCommandQueue) SeekNext() {
	c.seekBackOrFwd(1)
}

func (c *playbackCommandQueue) SeekBackOrPrevious() {
	c.seekBackOrFwd(-1)
}

func (c *playbackCommandQueue) UpdatePlayQueue(items []mediaprovider.MediaItem) {
	c.filterCommandsAndAdd([]playbackCommandType{cmdUpdatePlayQueue},
		playbackCommand{Type: cmdUpdatePlayQueue, Arg: items})
}

func (c *playbackCommandQueue) RemoveItemsFromQueue(idxs []int) {
	c.mutex.Lock()
	c.queue = append(c.queue, playbackCommand{
		Type: cmdRemoveTracksFromQueue,
		Arg:  idxs,
	})
	c.mutex.Unlock()
	c.cmdAvailable.Signal()
}

func (c *playbackCommandQueue) LoadRadioStation(radio *mediaprovider.RadioStation, insertMode InsertQueueMode) {
	c.mutex.Lock()
	c.queue = append(c.queue, playbackCommand{
		Type: cmdLoadRadioStation,
		Arg:  radio,
		Arg2: insertMode,
	})
	c.mutex.Unlock()
	c.cmdAvailable.Signal()
}

func (c *playbackCommandQueue) LoadItems(items []mediaprovider.MediaItem, insertQueueMode InsertQueueMode, shuffle bool) {
	c.mutex.Lock()
	c.queue = append(c.queue, playbackCommand{
		Type: cmdLoadItems,
		Arg:  items,
		Arg2: insertQueueMode,
		Arg3: shuffle,
	})
	c.mutex.Unlock()
	c.cmdAvailable.Signal()
}

func (c *playbackCommandQueue) addCommand(command playbackCommand) {
	c.mutex.Lock()
	c.queue = append(c.queue, command)
	c.mutex.Unlock()
	c.cmdAvailable.Signal()
}

func (c *playbackCommandQueue) filterCommandsAndAdd(excludeTypes []playbackCommandType, command playbackCommand) {
	c.mutex.Lock()
	j := 0
	for _, cmd := range c.queue {
		if slices.Contains(excludeTypes, cmd.Type) {
			continue
		}
		c.queue[j] = cmd
		j++
	}
	c.queue = c.queue[:j]
	c.queue = append(c.queue, command)
	c.mutex.Unlock()
	c.cmdAvailable.Signal()
}

func (c *playbackCommandQueue) seekBackOrFwd(direction int) {
	// find the index of the last seekBackOrFwd command
	// in the queue that can be coalesced with this one
	lastIdx := -1
	c.mutex.Lock()
	done := false
	for i := len(c.queue) - 1; i >= 0 && !done; i-- {
		cmd := c.queue[i]
		switch cmd.Type {
		case cmdSeekFwdBackN:
			lastIdx = i
		case cmdRemoveTracksFromQueue, cmdLoadItems, cmdPlayTrackAt,
			cmdLoadRadioStation, cmdUpdatePlayQueue, cmdStopAndClearPlayQueue:
			// any queue-modifying command means we can't coalesce any
			// more seekFwdBackN commands before here
			done = true
		}
	}

	if lastIdx == -1 {
		// no coalescable seekFwdBackN commands, just append new one
		c.queue = append(c.queue, playbackCommand{Type: cmdSeekFwdBackN, Arg: direction})
	} else {
		newQueue := make([]playbackCommand, 0, len(c.queue))
		// copy over all cmds past the first coalescable idx
		newQueue = append(newQueue, c.queue[0:lastIdx]...)
		n := direction
		for i := lastIdx; i < len(c.queue); i++ {
			if cmd := c.queue[i]; cmd.Type == cmdSeekFwdBackN {
				// coalesce this cmd with the new one
				n += cmd.Arg.(int)
			} else {
				// copy over other non-seekFwdBackN command
				newQueue = append(newQueue, cmd)
			}
		}
		newQueue = append(newQueue, playbackCommand{
			Type: cmdSeekFwdBackN,
			Arg:  n,
		})
		c.queue = newQueue
	}

	c.mutex.Unlock()
	c.cmdAvailable.Signal()
}

func (c *playbackCommandQueue) chanWriter() {
	for {
		c.mutex.Lock()
		for len(c.queue) == 0 {
			c.cmdAvailable.Wait()
		}
		cmd := c.queue[0]
		copy(c.queue, c.queue[1:])
		c.queue = c.queue[:len(c.queue)-1]
		c.mutex.Unlock()
		c.nextChan <- cmd
	}
}
