package dlna

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/backend/player"
	"github.com/dweymouth/supersonic/backend/util"
	"github.com/huin/goupnp/dcps/av1"
)

const (
	stopped = 0
	playing = 1
	paused  = 2

	// DLNA device initialization timing constants
	maxSeekRetries        = 5
	seekRetryInitialDelay = 400 * time.Millisecond
	seekRetryMaxDelay     = 2 * time.Second
	playbackSyncDelay     = 500 * time.Millisecond
	playbackSyncRetries   = 3
)

// MediaItem represents a media item to be played via DLNA
type MediaItem struct {
	URL         string
	Title       string
	ContentType string
	Seekable    bool
}

type proxyMapEntry struct {
	key string
	url string
}

type DLNAPlayer struct {
	player.BasePlayerCallbackImpl

	destroyed     bool
	cancelRequest context.CancelFunc

	avTransport   *av1.AVTransport1
	renderControl *av1.RenderingControl1

	state   int // stopped, playing, paused
	seeking bool

	metaLock      sync.Mutex
	curTrackMeta  mediaprovider.MediaItemMetadata
	nextTrackMeta mediaprovider.MediaItemMetadata

	// if true, report playback time 00:00
	// pending time sync with player after beginning playback
	pendingPlayStart bool
	// start playback position in seconds of the last seek/time sync
	lastStartTime int
	// how long the track has been playing since last time sync
	stopwatch util.Stopwatch

	proxyServer *http.Server
	proxyActive atomic.Bool
	localIP     string
	proxyPort   int

	pendingSeek     bool
	pendingSeekSecs float64

	// keep in order of most recently accessed at the end
	// that way the item in proxyURLs[0] can be kicked out
	// when adding a new URL to the proxy, since
	// only two will need to be active at any given time
	proxyURLs    [3]proxyMapEntry
	proxyURLLock sync.Mutex

	// If SetNextAVTransport fails (e.g. because the device
	// does not support the API/gapless), this flag is set
	// true, and the next firing of the track change timer
	// should clear it to false and use SetAVTransport
	// to begin playing the item in nextTrackMeta.
	failedToSetNext    bool
	unsetNextMediaItem *MediaItem

	timerActive atomic.Bool
	timer       *time.Timer
	resetChan   chan (time.Duration)
}

func NewDLNAPlayer(device *MediaRendererDevice) (*DLNAPlayer, error) {
	avt, err := device.NewAVTransportClient()
	if err != nil {
		return nil, err
	}
	rc, err := device.NewRenderingControlClient()
	if err != nil {
		return nil, err
	}

	// ping to test connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, _, _, err := avt.GetTransportInfoCtx(ctx, 0); err != nil {
		return nil, fmt.Errorf("failed to connect to %s", device.FriendlyName)
	}

	return &DLNAPlayer{
		avTransport:   avt,
		renderControl: rc,
		resetChan:     make(chan time.Duration),
	}, nil
}

func (d *DLNAPlayer) SetVolume(vol int) error {
	if d.destroyed {
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	d.cancelRequest = cancel
	defer cancel()
	return d.renderControl.SetVolumeCtx(ctx, 0, "Master", uint16(vol))
}

func (d *DLNAPlayer) GetVolume() int {
	if d.destroyed {
		return 0
	}
	ctx, cancel := context.WithCancel(context.Background())
	d.cancelRequest = cancel
	defer cancel()
	vol, _ := d.renderControl.GetVolumeCtx(ctx, 0, "Master")
	return int(vol)
}

func (d *DLNAPlayer) PlayFile(urlstr string, meta mediaprovider.MediaItemMetadata, startTime float64) error {
	if d.destroyed {
		return nil
	}

	d.ensureSetupProxy()

	d.metaLock.Lock()
	d.curTrackMeta = meta
	d.metaLock.Unlock()
	key := d.addURLToProxy(urlstr)

	media := &MediaItem{
		URL:         d.urlForItem(key),
		Title:       meta.Name,
		ContentType: meta.MIMEType,
		Seekable:    true,
	}

	if err := d.playAVTransportMedia(media); err != nil {
		return err
	}
	d.pendingPlayStart = true

	// Handle initial seek or playback time synchronization asynchronously
	// to avoid blocking the main player thread
	if startTime > 0 {
		go func() {
			// The DLNA device needs time to process the play command before accepting seek
			// Use retry logic in sendSeekCmd to handle devices with varying readiness times
			if err := d.sendSeekCmd(startTime); err != nil {
				log.Printf("Failed to seek to start position %v: %v", startTime, err)
				// Fall back to syncing from current position
				d.syncPlaybackTime()
			}
			d.pendingPlayStart = false
		}()
	} else {
		go func() {
			// Give the device a moment to start playback before syncing time
			time.Sleep(playbackSyncDelay)
			if !d.destroyed {
				d.syncPlaybackTime()
			}
			d.pendingPlayStart = false
		}()
	}
	d.state = playing
	remainingDur := meta.Duration - time.Duration(startTime)*time.Second
	d.setTrackChangeTimer(remainingDur)
	d.stopwatch.Reset()
	d.stopwatch.Start()
	d.lastStartTime = int(startTime)
	d.InvokeOnPlaying()
	d.InvokeOnTrackChange()
	if startTime > 0 {
		d.InvokeOnSeek()
	}

	return nil
}

func (d *DLNAPlayer) playAVTransportMedia(media *MediaItem) error {
	ctx, cancel := context.WithCancel(context.Background())
	d.cancelRequest = cancel
	defer cancel()

	metadata := buildDIDLMetadata(media)
	err := d.avTransport.SetAVTransportURICtx(ctx, 0, media.URL, metadata)
	if err != nil {
		return err
	}
	if err := d.avTransport.PlayCtx(ctx, 0, "1"); err != nil {
		return err
	}
	return nil
}

func (d *DLNAPlayer) SetNextFile(url string, meta mediaprovider.MediaItemMetadata) error {
	if d.destroyed {
		return nil
	}

	var media *MediaItem
	d.metaLock.Lock()
	d.nextTrackMeta = meta
	d.metaLock.Unlock()
	if url != "" {
		d.ensureSetupProxy()

		key := d.addURLToProxy(url)
		media = &MediaItem{
			URL:         d.urlForItem(key),
			ContentType: meta.MIMEType,
			Title:       meta.Name,
			Seekable:    true,
		}
	} else {
		// empty media item to signify erasing next track in device queue
		media = &MediaItem{}
	}

	ctx, cancel := context.WithCancel(context.Background())
	d.cancelRequest = cancel
	defer cancel()

	metadata := buildDIDLMetadata(media)
	err := d.avTransport.SetNextAVTransportURICtx(ctx, 0, media.URL, metadata)
	if err != nil {
		d.metaLock.Lock()
		d.failedToSetNext = true
		d.unsetNextMediaItem = media
		d.metaLock.Unlock()
	}
	return err
}

func (d *DLNAPlayer) Continue() error {
	if d.destroyed || d.state == playing {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	d.cancelRequest = cancel
	defer cancel()

	if d.pendingSeek {
		d.pendingSeek = false
		target := formatTime(int(d.pendingSeekSecs))
		err := d.avTransport.SeekCtx(ctx, 0, "REL_TIME", target)
		if err != nil {
			return err
		}
	}

	if err := d.avTransport.PlayCtx(ctx, 0, "1"); err != nil {
		return err
	}
	d.metaLock.Lock()
	nextTrackChange := d.curTrackMeta.Duration - d.curPlayPos()
	d.metaLock.Unlock()
	d.state = playing
	d.setTrackChangeTimer(nextTrackChange)
	d.stopwatch.Start()
	d.InvokeOnPlaying()
	return nil
}

func (d *DLNAPlayer) Pause() error {
	if d.destroyed || d.state != playing {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	d.cancelRequest = cancel
	defer cancel()
	if err := d.avTransport.PauseCtx(ctx, 0); err != nil {
		return err
	}
	d.setTrackChangeTimer(0)
	d.stopwatch.Stop()
	d.state = paused
	d.InvokeOnPaused()
	return nil
}

func (d *DLNAPlayer) Stop(force bool) error {
	if d.destroyed {
		return nil
	}
	if force && d.cancelRequest != nil {
		d.cancelRequest()
	}

	switch d.state {
	case stopped:
		return nil
	case playing:
		var ctx context.Context
		var cancel context.CancelFunc
		if force {
			ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
		} else {
			ctx, cancel = context.WithCancel(context.Background())
		}
		d.cancelRequest = cancel
		defer cancel()

		if err := d.avTransport.PauseCtx(ctx, 0); err != nil {
			return err
		}
		fallthrough
	case paused:
		d.setTrackChangeTimer(0)
		d.stopwatch.Reset()
		d.lastStartTime = 0
		d.state = stopped
		d.InvokeOnStopped()
		return nil
	default:
		return errors.New("invalid player state")
	}
}

func (d *DLNAPlayer) SeekSeconds(secs float64) error {
	if d.destroyed {
		return nil
	}

	if d.state == paused {
		d.pendingSeek = true
		d.pendingSeekSecs = secs
	} else {
		if err := d.sendSeekCmd(secs); err != nil {
			return err
		}
	}

	d.lastStartTime = int(secs)
	d.stopwatch.Reset()

	if d.state == playing {
		d.metaLock.Lock()
		nextTrackChange := d.curTrackMeta.Duration - time.Duration(secs)*time.Second
		d.metaLock.Unlock()
		d.setTrackChangeTimer(nextTrackChange)
		d.stopwatch.Start()
	}

	d.InvokeOnSeek()

	go func() {
		time.Sleep(4 * time.Second)
		if !d.destroyed {
			d.syncPlaybackTime()
		}
	}()
	return nil
}

func (d *DLNAPlayer) sendSeekCmd(secs float64) error {
	d.seeking = true
	defer func() { d.seeking = false }()

	// Retry seeking with exponential backoff to handle devices that aren't
	// immediately ready after playback starts
	delay := seekRetryInitialDelay
	var lastErr error

	for attempt := 0; attempt < maxSeekRetries; attempt++ {
		if d.destroyed {
			return errors.New("player destroyed during seek")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		target := formatTime(int(secs))
		err := d.avTransport.SeekCtx(ctx, 0, "REL_TIME", target)
		cancel()

		if err == nil {
			return nil
		}

		lastErr = err
		log.Printf("DLNA seek attempt %d/%d failed: %v, retrying in %v",
			attempt+1, maxSeekRetries, err, delay)

		time.Sleep(delay)
		// Exponential backoff, but cap at max delay
		delay *= 2
		if delay > seekRetryMaxDelay {
			delay = seekRetryMaxDelay
		}
	}

	return fmt.Errorf("failed to seek after %d attempts: %w", maxSeekRetries, lastErr)
}

func (d *DLNAPlayer) IsSeeking() bool {
	return d.seeking
}

func (d *DLNAPlayer) GetStatus() player.Status {
	state := player.Stopped
	if d.state == playing {
		state = player.Playing
	} else if d.state == paused {
		state = player.Paused
	}

	var timePos float64
	if !d.pendingPlayStart {
		timePos = d.curPlayPos().Seconds()
	}
	return player.Status{
		State:    state,
		TimePos:  timePos,
		Duration: d.curTrackMeta.Duration.Seconds(),
	}
}

func (d *DLNAPlayer) curPlayPos() time.Duration {
	return time.Duration(d.lastStartTime)*time.Second + d.stopwatch.Elapsed()
}

func (d *DLNAPlayer) Destroy() {
	d.destroyed = true
	d.setTrackChangeTimer(0)
	if d.cancelRequest != nil {
		d.cancelRequest()
	}

	if d.proxyServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		go d.proxyServer.Shutdown(ctx)
		d.proxyServer = nil
	}
}

func (d *DLNAPlayer) syncPlaybackTime() {
	// Retry playback time synchronization with the DLNA device
	var lastErr error
	for attempt := 0; attempt < playbackSyncRetries; attempt++ {
		if d.destroyed {
			return
		}

		start := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		_, relTime, _, _, _, _, _, _, err := d.avTransport.GetPositionInfoCtx(ctx, 0)
		cancel()

		if err == nil {
			// Parse the RelTime string (format: HH:MM:SS or HH:MM:SS.mmm)
			var hours, minutes, seconds int
			_, parseErr := fmt.Sscanf(relTime, "%d:%d:%d", &hours, &minutes, &seconds)
			if parseErr != nil {
				log.Printf("Failed to parse position time '%s': %v", relTime, parseErr)
				lastErr = parseErr
				continue
			}

			// Account for network latency by adding half the round-trip time
			networkLatency := time.Since(start) / 2
			posSeconds := hours*3600 + minutes*60 + seconds
			d.lastStartTime = int(float64(posSeconds) + networkLatency.Seconds())
			d.stopwatch.Reset()
			if d.state == playing {
				d.stopwatch.Start()
			}
			d.setTrackChangeTimer(d.curTrackMeta.Duration - time.Duration(d.lastStartTime)*time.Second)
			d.InvokeOnSeek()
			return
		}

		lastErr = err
		if attempt < playbackSyncRetries-1 {
			log.Printf("DLNA playback sync attempt %d/%d failed: %v, retrying",
				attempt+1, playbackSyncRetries, err)
			time.Sleep(playbackSyncDelay)
		}
	}

	// Log final failure but don't crash - player can still work without perfect sync
	log.Printf("Failed to sync playback time after %d attempts: %v", playbackSyncRetries, lastErr)
}

func (d *DLNAPlayer) ensureSetupProxy() error {
	if d.proxyActive.Swap(true) {
		return nil // already active
	}

	var err error
	d.localIP, err = util.GetLocalIP()
	if err != nil {
		return err
	}

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return err
	}
	d.proxyPort = listener.Addr().(*net.TCPAddr).Port

	d.proxyServer = &http.Server{
		Handler: http.HandlerFunc(d.handleRequest),
	}

	go d.proxyServer.Serve(listener)
	return nil
}

func (d *DLNAPlayer) setTrackChangeTimer(dur time.Duration) {
	if d.timerActive.Swap(true) {
		// was active
		d.resetChan <- dur
		return
	}
	if dur == 0 {
		d.timerActive.Store(false)
		return
	}

	d.timer = time.NewTimer(dur)
	go func() {
		for {
			select {
			case dur := <-d.resetChan:
				if dur == 0 {
					d.timerActive.Store(false)
					if !d.timer.Stop() {
						select {
						case <-d.timer.C:
						default:
						}
					}
					d.timer = nil
					return
				}
				// reset the timer
				if !d.timer.Stop() {
					select {
					case <-d.timer.C:
					default:
					}
				}
				d.timer.Reset(dur)
			case <-d.timer.C:
				d.timerActive.Store(false)
				d.timer = nil
				d.handleOnTrackChange()
				return
			}
		}
	}()
}

func (d *DLNAPlayer) handleOnTrackChange() {
	stopping := false
	d.metaLock.Lock()
	if d.nextTrackMeta.ID == "" {
		stopping = true
	}
	d.curTrackMeta = d.nextTrackMeta
	d.nextTrackMeta = mediaprovider.MediaItemMetadata{}
	nextTrackChange := d.curTrackMeta.Duration
	d.metaLock.Unlock()

	if stopping {
		d.lastStartTime = 0
		d.stopwatch.Reset()
		d.InvokeOnStopped()
	} else {
		d.metaLock.Lock()
		if d.failedToSetNext {
			d.failedToSetNext = false
			media := d.unsetNextMediaItem
			d.unsetNextMediaItem = nil
			d.metaLock.Unlock()
			d.playAVTransportMedia(media)
		} else {
			d.metaLock.Unlock()
		}

		d.lastStartTime = 0
		d.stopwatch.Reset()
		d.stopwatch.Start()
		d.setTrackChangeTimer(nextTrackChange)
		d.InvokeOnTrackChange()

		go func() {
			time.Sleep(5 * time.Second)
			if !d.destroyed {
				d.syncPlaybackTime()
			}
		}()
	}
}

func (d *DLNAPlayer) urlForItem(key string) string {
	return fmt.Sprintf("http://%s:%d/%s", d.localIP, d.proxyPort, key)
}

func (d *DLNAPlayer) handleRequest(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/")
	url, _ := d.lookupProxyURL(key)

	if url == "" {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("404"))
		return
	}

	// if the url is a filepath for a local cached file, serve it
	if info, err := os.Stat(url); err == nil && info.Size() > 0 {
		http.ServeFile(w, r, url)
		return
	}

	// Otherwise, proxy request to the music server
	proxyReq, err := http.NewRequest(r.Method, url, r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Copy headers from the original request to the new request
	proxyReq.Header = r.Header

	// Create an HTTP client and send the request
	client := &http.Client{}
	resp, err := client.Do(proxyReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy headers from the response to the writer
	for name, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}

	// Set the status code
	w.WriteHeader(resp.StatusCode)

	// Copy the response body to the writer
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error copying response body:", err)
	}
}

func (d *DLNAPlayer) addURLToProxy(url string) string {
	hash := md5.Sum([]byte(url))
	key := base64.StdEncoding.EncodeToString(hash[:])
	d.proxyURLLock.Lock()
	defer d.proxyURLLock.Unlock()
	d._updateProxyURL(key, url)
	return key
}

// lookupProxyURL finds a URL by key and updates its position to most recently used
func (d *DLNAPlayer) lookupProxyURL(key string) (string, bool) {
	d.proxyURLLock.Lock()
	defer d.proxyURLLock.Unlock()

	for i := range len(d.proxyURLs) {
		if d.proxyURLs[i].key == key {
			url := d.proxyURLs[i].url
			// Move accessed entry to the most recent position
			d._updateProxyURL(key, url)
			return url, true
		}
	}

	return "", false
}

func (d *DLNAPlayer) _updateProxyURL(key, url string) {
	// Check if the key already exists, and if so, move it to the most recently used position
	for i := range len(d.proxyURLs) {
		if d.proxyURLs[i].key == key {
			if i < len(d.proxyURLs)-1 {
				// Shift elements to the left from found position to the end
				copy(d.proxyURLs[i:], d.proxyURLs[i+1:])
			}
			// Place updated entry at the last position
			d.proxyURLs[len(d.proxyURLs)-1] = proxyMapEntry{key: key, url: url}
			return
		}
	}

	// Shift all elements left to make room for the new entry at the end
	copy(d.proxyURLs[:], d.proxyURLs[1:])
	// Insert new element at the most recent position
	d.proxyURLs[len(d.proxyURLs)-1] = proxyMapEntry{key: key, url: url}
}

// formatTime converts seconds to DLNA time format (HH:MM:SS)
func formatTime(seconds int) string {
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, secs)
}

// buildDIDLMetadata creates DIDL-Lite XML metadata for a media item
func buildDIDLMetadata(media *MediaItem) string {
	if media == nil || media.URL == "" {
		return ""
	}
	return fmt.Sprintf(`<DIDL-Lite xmlns="urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:upnp="urn:schemas-upnp-org:metadata-1-0/upnp/">
<item id="0" parentID="-1" restricted="1">
<dc:title>%s</dc:title>
<res protocolInfo="http-get:*:%s:*">%s</res>
<upnp:class>object.item.audioItem.musicTrack</upnp:class>
</item>
</DIDL-Lite>`, media.Title, media.ContentType, media.URL)
}
