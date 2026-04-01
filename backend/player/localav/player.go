// Package localav implements a URLPlayer backed by FFmpeg (libav*) for
// demuxing/decoding/filtering and miniaudio for audio output.
// It provides direct access to decoded PCM samples for visualisers such as
// ProjectM, and supports gapless playback, device selection, exclusive mode,
// a parametric EQ, and ReplayGain.
package localav

/*
#include "av_player.h"
#include <stdlib.h>
*/
import "C"

import (
	"math"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/backend/player"
)

// AudioDevice and MediaInfo are defined in the parent player package.

var _ player.LocalPlayer = (*Player)(nil)

// Player is the localav URLPlayer implementation.
type Player struct {
	player.BasePlayerCallbackImpl

	ctx     *C.av_player_t
	mu      sync.Mutex // guards filter state, nextURL fields
	initd   bool

	vol            int
	audioExclusive bool
	pauseFade      bool

	equalizer      player.Equalizer
	replayGainOpts player.ReplayGainOptions
	peaksEnabled   bool

	// next-track state (mirrors what we've sent to C via av_player_open_next)
	nextURL          string
	nextEQFilter     string
	nextRGGainDB     float64
	nextRGPreventClip int

	status player.Status

	// decode loop control
	stopDecode    chan struct{} // closed to signal the loop to exit
	decodeStopped chan struct{} // closed when the loop has exited
}

// New creates a new Player.  Call Init before use.
func New() *Player {
	return &Player{
		vol: 100,
	}
}

// Init allocates the C player and opens the default audio device.
func (p *Player) Init() error {
	p.ctx = C.av_player_create()
	if p.ctx == nil {
		return errorf("av_player_create failed")
	}
	if ret := C.av_player_init(p.ctx, nil, 0); ret != 0 {
		C.av_player_destroy(p.ctx)
		p.ctx = nil
		return errorf("av_player_init failed: %d", int(ret))
	}
	p.initd = true
	p.setVolumeCGo(p.vol)
	return nil
}

// ---- URLPlayer -------------------------------------------------------

func (p *Player) PlayFile(url string, _ mediaprovider.MediaItemMetadata, startTime float64) error {
	if !p.initd {
		return ErrUnitialized
	}
	p.stopDecodeLoop()

	eqFilter := p.buildEQFilter()
	rgGain, rgClip := p.replayGainParams()

	curl := C.CString(url)
	ceq := C.CString(eqFilter)
	defer C.free(unsafe.Pointer(curl))
	defer C.free(unsafe.Pointer(ceq))

	ret := C.av_player_open(p.ctx, curl, C.double(startTime), ceq,
		C.double(rgGain), C.int(rgClip))
	if ret != 0 {
		return errorf("av_player_open failed: %d", int(ret))
	}

	p.status.State = player.Playing
	p.startDecodeLoop()
	p.InvokeOnTrackChange()
	p.InvokeOnPlaying()
	return nil
}

func (p *Player) SetNextFile(url string, _ mediaprovider.MediaItemMetadata) error {
	if !p.initd {
		return ErrUnitialized
	}
	p.mu.Lock()
	p.nextURL = url
	p.nextEQFilter = p.buildEQFilter()
	p.nextRGGainDB, p.nextRGPreventClip = p.replayGainParams()
	p.mu.Unlock()

	eqFilter := p.nextEQFilter
	rgGain := p.nextRGGainDB
	rgClip := p.nextRGPreventClip

	curl := C.CString(url)
	ceq := C.CString(eqFilter)
	defer C.free(unsafe.Pointer(curl))
	defer C.free(unsafe.Pointer(ceq))

	ret := C.av_player_open_next(p.ctx, curl, ceq, C.double(rgGain), C.int(rgClip))
	if ret != 0 {
		return errorf("av_player_open_next failed: %d", int(ret))
	}
	return nil
}

// ---- BasePlayer ------------------------------------------------------

func (p *Player) Continue() error {
	if p.status.State != player.Paused {
		return nil
	}
	C.av_player_resume(p.ctx)
	p.status.State = player.Playing
	p.InvokeOnPlaying()
	return nil
}

func (p *Player) Pause() error {
	if p.status.State != player.Playing {
		return nil
	}
	if p.pauseFade {
		go p.fadeAndPause()
		return nil
	}
	C.av_player_pause(p.ctx)
	p.status.State = player.Paused
	p.InvokeOnPaused()
	return nil
}

func (p *Player) Stop(_ bool) error {
	if !p.initd {
		return ErrUnitialized
	}
	p.stopDecodeLoop()
	C.av_player_stop(p.ctx)
	p.status.State = player.Stopped
	p.status.TimePos = 0
	p.status.Duration = 0
	p.InvokeOnStopped()
	return nil
}

func (p *Player) SeekSeconds(secs float64) error {
	if !p.initd {
		return ErrUnitialized
	}
	// Stop the decode loop so it doesn't race with the filter graph rebuild inside seek.
	wasPlaying := p.status.State == player.Playing || p.status.State == player.Paused
	p.stopDecodeLoop()
	C.av_player_seek(p.ctx, C.double(secs))
	p.InvokeOnSeek()
	if wasPlaying {
		p.startDecodeLoop()
	}
	return nil
}

func (p *Player) IsSeeking() bool {
	return false // libav seek is synchronous from Go's perspective
}

func (p *Player) SetVolume(vol int) error {
	if vol > 100 {
		vol = 100
	} else if vol < 0 {
		vol = 0
	}
	p.vol = vol
	if p.initd {
		p.setVolumeCGo(vol)
	}
	return nil
}

func (p *Player) GetVolume() int {
	return p.vol
}

func (p *Player) GetStatus() player.Status {
	if !p.initd {
		return p.status
	}
	p.status.TimePos = float64(C.av_player_get_position(p.ctx))
	p.status.Duration = float64(C.av_player_get_duration(p.ctx))
	state := int(C.av_player_get_state(p.ctx))
	switch state {
	case C.AVPLAYER_STATE_PLAYING:
		p.status.State = player.Playing
	case C.AVPLAYER_STATE_PAUSED:
		p.status.State = player.Paused
	default:
		p.status.State = player.Stopped
	}
	return p.status
}

func (p *Player) Destroy() {
	p.stopDecodeLoop()
	if p.ctx != nil {
		C.av_player_destroy(p.ctx)
		p.ctx = nil
	}
	p.initd = false
}

// ---- ReplayGainPlayer ------------------------------------------------

func (p *Player) SetReplayGainOptions(opts player.ReplayGainOptions) error {
	p.replayGainOpts = opts
	return p.rebuildFilters()
}

// ---- Extended API (matching mpv.Player extras used by UI/controller) ---

func (p *Player) SetEqualizer(eq player.Equalizer) error {
	p.equalizer = eq
	return p.rebuildFilters()
}

func (p *Player) Equalizer() player.Equalizer {
	return p.equalizer
}

func (p *Player) SetPauseFade(enabled bool) {
	p.pauseFade = enabled
}

func (p *Player) SetAudioExclusive(exclusive bool) {
	p.audioExclusive = exclusive
	if p.initd {
		excl := 0
		if exclusive {
			excl = 1
		}
		C.av_player_set_exclusive(p.ctx, C.int(excl))
	}
}

func (p *Player) ListAudioDevices() ([]player.AudioDevice, error) {
	const maxDevices = 64
	cdevices := make([]C.av_device_info_t, maxDevices)
	count := int(C.av_player_list_devices(&cdevices[0], C.int(maxDevices)))
	devices := make([]player.AudioDevice, count)
	for i := range devices {
		devices[i].Name = C.GoString(&cdevices[i].name[0])
		devices[i].Description = C.GoString(&cdevices[i].description[0])
	}
	return devices, nil
}

func (p *Player) SetAudioDevice(deviceName string) error {
	if !p.initd {
		return ErrUnitialized
	}
	cname := C.CString(deviceName)
	defer C.free(unsafe.Pointer(cname))
	ret := C.av_player_set_device(p.ctx, cname)
	if ret != 0 {
		return errorf("SetAudioDevice failed: %d", int(ret))
	}
	return nil
}

func (p *Player) GetMediaInfo() (player.MediaInfo, error) {
	var info player.MediaInfo
	if !p.initd {
		return info, ErrUnitialized
	}
	var cinfo C.av_media_info_t
	C.av_player_get_media_info(p.ctx, &cinfo)
	info.Codec = C.GoString(&cinfo.codec[0])
	info.Samplerate = int(cinfo.sample_rate)
	info.ChannelCount = int(cinfo.channels)
	info.Bitrate = int(cinfo.bitrate)
	return info, nil
}

// SetPeaksEnabled enables/disables the astats filter for peak metering.
func (p *Player) SetPeaksEnabled(enabled bool) error {
	p.peaksEnabled = enabled
	if p.initd {
		v := 0
		if enabled {
			v = 1
		}
		C.av_player_set_peaks_enabled(p.ctx, C.int(v))
	}
	return nil
}

// GetPeaks returns the latest peak/RMS values in dBFS.
// Returns -Inf for all channels when not playing or peaks not enabled.
func (p *Player) GetPeaks() (float64, float64, float64, float64) {
	nInf := math.Inf(-1)
	if !p.initd || p.status.State != player.Playing {
		return nInf, nInf, nInf, nInf
	}
	var lp, rp, lr, rr C.double
	C.av_player_get_peaks(p.ctx, &lp, &rp, &lr, &rr)
	return float64(lp), float64(rp), float64(lr), float64(rr)
}

// ObserveIcyRadioTitle is a no-op stub; ICY metadata is not yet implemented
// in the localav backend.
func (p *Player) ObserveIcyRadioTitle(_ func(string)) {}

// UnobserveIcyRadioTitle is a no-op stub.
func (p *Player) UnobserveIcyRadioTitle() {}

// ---- Internal helpers ------------------------------------------------

func (p *Player) setVolumeCGo(vol int) {
	C.av_player_set_volume(p.ctx, C.float(float32(vol)/100.0))
}

func (p *Player) buildEQFilter() string {
	if p.equalizer == nil || !p.equalizer.IsEnabled() {
		return ""
	}
	var parts []string
	if math.Abs(p.equalizer.Preamp()) > 0.01 {
		// Preamp handled via rg_gain_db offset — include it in the filter string
		// so it's distinct from the ReplayGain volume adjustment
	}
	if s := p.equalizer.Curve().String(); s != "" {
		parts = append(parts, s)
	}
	return strings.Join(parts, ",")
}

// replayGainParams returns the total volume offset (preamp + RG mode) and clip flag.
func (p *Player) replayGainParams() (gainDB float64, preventClip int) {
	gainDB = 0
	if p.equalizer != nil && p.equalizer.IsEnabled() {
		gainDB += p.equalizer.Preamp()
	}
	// Note: actual per-track RG gain is read from file tags by the C layer
	// when rg_gain_db is 0.  For now we pass the preamp only.
	if p.replayGainOpts.PreventClipping {
		preventClip = 1
	}
	return
}

func (p *Player) rebuildFilters() error {
	if !p.initd || p.ctx == nil {
		return nil
	}
	eqFilter := p.buildEQFilter()
	rgGain, rgClip := p.replayGainParams()
	ceq := C.CString(eqFilter)
	defer C.free(unsafe.Pointer(ceq))
	ret := C.av_player_set_filters(p.ctx, ceq, C.double(rgGain), C.int(rgClip))
	if ret != 0 {
		return errorf("set_filters failed: %d", int(ret))
	}
	return nil
}

func (p *Player) fadeAndPause() {
	vol := p.vol
	for c := 0; c < 100; c++ {
		newVol := float32(vol) * float32(100-c) / 100.0
		C.av_player_set_volume(p.ctx, C.float(newVol/100.0))
		time.Sleep(2 * time.Millisecond)
	}
	C.av_player_pause(p.ctx)
	p.setVolumeCGo(p.vol)
	p.status.State = player.Paused
	p.InvokeOnPaused()
}

// ---- Decode loop (goroutine) -----------------------------------------

func (p *Player) startDecodeLoop() {
	p.stopDecode = make(chan struct{})
	p.decodeStopped = make(chan struct{})
	go func() {
		defer close(p.decodeStopped)
		p.decodeLoop(p.stopDecode)
	}()
}

func (p *Player) stopDecodeLoop() {
	if p.stopDecode == nil {
		return
	}
	select {
	case <-p.stopDecode:
		// already closed
	default:
		close(p.stopDecode)
	}
	<-p.decodeStopped
	p.stopDecode = nil
	p.decodeStopped = nil
}

func (p *Player) decodeLoop(stop <-chan struct{}) {
	for {
		select {
		case <-stop:
			return
		default:
		}

		result := int(C.av_player_decode_step(p.ctx))

		switch result {
		case C.AVPLAYER_DECODE_OK:
			// Good — keep going without sleeping

		case C.AVPLAYER_DECODE_RING_FULL:
			// Ring buffer full or paused — back off
			select {
			case <-stop:
				return
			case <-time.After(10 * time.Millisecond):
			}

		case C.AVPLAYER_DECODE_EOF:
			// Current track done; ring is draining — loop back to RING_FULL handling
			select {
			case <-stop:
				return
			case <-time.After(5 * time.Millisecond):
			}

		case C.AVPLAYER_DECODE_NEXT_READY:
			// Gapless: next track was swapped in
			p.InvokeOnTrackChange()

		case C.AVPLAYER_DECODE_STOPPED:
			// Ring drained, no next track — truly stopped
			if p.status.State != player.Stopped {
				p.status.State = player.Stopped
				p.InvokeOnStopped()
			}
			return

		case C.AVPLAYER_DECODE_ERROR:
			p.status.State = player.Stopped
			p.InvokeOnStopped()
			return
		}
	}
}
