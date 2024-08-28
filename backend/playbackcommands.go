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
)

type PlaybackCommand struct {
	Type playbackCommandType
	Arg  any
	Arg2 any
	Arg3 any
}

type CommandQueue struct {
	mutex        sync.Mutex
	queue        []PlaybackCommand
	cmdAvailable *sync.Cond
	nextChan     chan (PlaybackCommand)
}

func NewCommandQueue() *CommandQueue {
	c := &CommandQueue{}
	c.cmdAvailable = sync.NewCond(&c.mutex)
	go c.chanWriter()
	return c
}

func (c *CommandQueue) C() <-chan PlaybackCommand {
	return c.nextChan
}

func (c *CommandQueue) Stop() {
	c.filterCommandsAndAdd([]playbackCommandType{cmdContinue, cmdPause, cmdStop},
		PlaybackCommand{Type: cmdStop})
}

func (c *CommandQueue) Continue() {
	c.filterCommandsAndAdd([]playbackCommandType{cmdContinue, cmdPause, cmdStop},
		PlaybackCommand{Type: cmdContinue})
}

func (c *CommandQueue) Pause() {
	c.filterCommandsAndAdd([]playbackCommandType{cmdContinue, cmdPause, cmdStop},
		PlaybackCommand{Type: cmdPause})
}

func (c *CommandQueue) StopAndClearPlayQueue() {
	c.filterCommandsAndAdd([]playbackCommandType{cmdContinue, cmdPause, cmdStop, cmdStopAndClearPlayQueue},
		PlaybackCommand{Type: cmdStopAndClearPlayQueue})
}

func (c *CommandQueue) SetVolume(vol int) {
	c.filterCommandsAndAdd([]playbackCommandType{cmdVolume},
		PlaybackCommand{Type: cmdVolume, Arg: vol})
}

func (c *CommandQueue) SetLoopMode(mode LoopMode) {
	c.filterCommandsAndAdd([]playbackCommandType{cmdLoopMode},
		PlaybackCommand{Type: cmdLoopMode, Arg: mode})
}

func (c *CommandQueue) SeekSeconds(s float64) {
	c.filterCommandsAndAdd([]playbackCommandType{cmdSeekSeconds},
		PlaybackCommand{Type: cmdSeekSeconds, Arg: s})
}

func (c *CommandQueue) SeekNext() {
	c.seekBackOrFwd(1)
}

func (c *CommandQueue) SeekBackOrPrevious() {
	c.seekBackOrFwd(-1)
}

func (c *CommandQueue) UpdatePlayQueue(items []mediaprovider.MediaItem) {
	c.filterCommandsAndAdd([]playbackCommandType{cmdUpdatePlayQueue},
		PlaybackCommand{Type: cmdUpdatePlayQueue, Arg: items})
}

func (c *CommandQueue) RemoveItemsFromQueue(idxs []int) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.queue = append(c.queue, PlaybackCommand{
		Type: cmdRemoveTracksFromQueue,
		Arg:  idxs,
	})
}

func (c *CommandQueue) LoadRadioStation(radio *mediaprovider.RadioStation, insertMode InsertQueueMode) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.queue = append(c.queue, PlaybackCommand{
		Type: cmdLoadRadioStation,
		Arg:  radio,
		Arg2: insertMode,
	})
}

func (c *CommandQueue) LoadItems(items []mediaprovider.MediaItem, insertQueueMode InsertQueueMode, shuffle bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.queue = append(c.queue, PlaybackCommand{
		Type: cmdLoadItems,
		Arg:  items,
		Arg2: insertQueueMode,
		Arg3: shuffle,
	})
}

func (c *CommandQueue) filterCommandsAndAdd(excludeTypes []playbackCommandType, command PlaybackCommand) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

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
}

func (c *CommandQueue) seekBackOrFwd(direction int) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	j := 0
	n := 0
	for _, cmd := range c.queue {
		if cmd.Type == cmdSeekFwdBackN {
			n += cmd.Arg.(int)
		} else {
			c.queue[j] = cmd
			j++
		}
	}
	c.queue = c.queue[:j]
	c.queue = append(c.queue, PlaybackCommand{
		Type: cmdSeekFwdBackN,
		Arg:  n + direction})
}

func (c *CommandQueue) chanWriter() {
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
