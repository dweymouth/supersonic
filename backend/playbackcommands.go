package backend

import (
	"slices"
	"sync"
)

type PlaybackCommandType int

const (
	CmdStop PlaybackCommandType = iota
	CmdContinue
	CmdPause
	CmdPlayTrackAt
	CmdSeekSeconds
	CmdSeekFwdBackN
	CmdVolume
	CmdLoopMode
)

type PlaybackCommand struct {
	Type PlaybackCommandType
	Arg  any
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
	c.filterCommandsAndAdd([]PlaybackCommandType{CmdContinue, CmdPause, CmdStop},
		PlaybackCommand{Type: CmdStop})
}

func (c *CommandQueue) Continue() {
	c.filterCommandsAndAdd([]PlaybackCommandType{CmdContinue, CmdPause, CmdStop},
		PlaybackCommand{Type: CmdContinue})
}

func (c *CommandQueue) Pause() {
	c.filterCommandsAndAdd([]PlaybackCommandType{CmdContinue, CmdPause, CmdStop},
		PlaybackCommand{Type: CmdPause})
}

func (c *CommandQueue) Volume(vol int) {
	c.filterCommandsAndAdd([]PlaybackCommandType{CmdVolume},
		PlaybackCommand{Type: CmdVolume, Arg: vol})
}

func (c *CommandQueue) LoopMode(mode LoopMode) {
	c.filterCommandsAndAdd([]PlaybackCommandType{CmdLoopMode},
		PlaybackCommand{Type: CmdLoopMode, Arg: mode})
}

func (c *CommandQueue) SeekSeconds(s float64) {
	c.filterCommandsAndAdd([]PlaybackCommandType{CmdVolume},
		PlaybackCommand{Type: CmdSeekSeconds, Arg: s})
}

func (c *CommandQueue) filterCommandsAndAdd(excludeTypes []PlaybackCommandType, command PlaybackCommand) {
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
