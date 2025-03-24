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
	"github.com/hashicorp/go-retryablehttp"
	"github.com/supersonic-app/go-upnpcast/device"
	"github.com/supersonic-app/go-upnpcast/services/avtransport"
	"github.com/supersonic-app/go-upnpcast/services/renderingcontrol"
)

const (
	stopped = 0
	playing = 1
	paused  = 2
)

var unimplemented = errors.New("unimplemented")

type proxyMapEntry struct {
	key string
	url string
}

type DLNAPlayer struct {
	player.BasePlayerCallbackImpl

	avTransport   *avtransport.Client
	renderControl *renderingcontrol.Client

	state   int // stopped, playing, paused
	seeking bool

	metaLock      sync.Mutex
	curTrackMeta  mediaprovider.MediaItemMetadata
	nextTrackMeta mediaprovider.MediaItemMetadata

	lastStartTime int
	stopwatch     util.Stopwatch

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
	unsetNextMediaItem *avtransport.MediaItem

	timerActive atomic.Bool
	timer       *time.Timer
	resetChan   chan (time.Duration)
}

func NewDLNAPlayer(device *device.MediaRenderer) (*DLNAPlayer, error) {
	retry := retryablehttp.NewClient()
	retry.RetryMax = 3
	retry.RetryWaitMin = 100 * time.Millisecond
	retry.Logger = retryLogger{}
	cli := retry.StandardClient()

	avt, err := device.AVTransportClient()
	if err != nil {
		return nil, err
	}
	avt.HTTPClient = cli
	rc, err := device.RenderingControlClient()
	if err != nil {
		return nil, err
	}
	rc.HTTPClient = cli
	return &DLNAPlayer{
		avTransport:   avt,
		renderControl: rc,
		resetChan:     make(chan time.Duration),
	}, nil
}

func (d *DLNAPlayer) SetVolume(vol int) error {
	return d.renderControl.SetVolume(context.Background(), vol)
}

func (d *DLNAPlayer) GetVolume() int {
	vol, _ := d.renderControl.GetVolume(context.Background())
	return vol
}

func (d *DLNAPlayer) PlayFile(urlstr string, meta mediaprovider.MediaItemMetadata) error {
	d.ensureSetupProxy()

	d.metaLock.Lock()
	d.curTrackMeta = meta
	d.metaLock.Unlock()
	key := d.addURLToProxy(urlstr)

	media := avtransport.MediaItem{
		URL:   d.urlForItem(key),
		Title: meta.Name,
	}

	if err := d.playAVTransportMedia(&media); err != nil {
		return err
	}
	d.state = playing
	d.setTrackChangeTimer(time.Duration(meta.Duration) * time.Second)
	d.stopwatch.Reset()
	d.stopwatch.Start()
	d.lastStartTime = 0
	d.InvokeOnPlaying()
	d.InvokeOnTrackChange()
	return nil
}

func (d *DLNAPlayer) playAVTransportMedia(media *avtransport.MediaItem) error {
	err := d.avTransport.SetAVTransportMedia(context.Background(), media)
	if err != nil {
		return err
	}
	if err := d.avTransport.Play(context.Background()); err != nil {
		return err
	}
	return nil
}

func (d *DLNAPlayer) SetNextFile(url string, meta mediaprovider.MediaItemMetadata) error {
	var media *avtransport.MediaItem
	d.metaLock.Lock()
	d.nextTrackMeta = meta
	d.metaLock.Unlock()
	if url != "" {
		d.ensureSetupProxy()

		key := d.addURLToProxy(url)
		media = &avtransport.MediaItem{
			URL:   d.urlForItem(key),
			Title: meta.Name,
		}
	}
	err := d.avTransport.SetNextAVTransportMedia(context.Background(), media)
	if err != nil {
		d.metaLock.Lock()
		d.failedToSetNext = true
		d.unsetNextMediaItem = media
		d.metaLock.Unlock()
	}
	return err
}

func (d *DLNAPlayer) Continue() error {
	if d.state == playing {
		return nil
	}

	if d.pendingSeek {
		d.pendingSeek = false
		err := d.avTransport.Seek(context.Background(), int(d.pendingSeekSecs))
		if err != nil {
			return err
		}
	}

	if err := d.avTransport.Play(context.Background()); err != nil {
		return err
	}
	d.metaLock.Lock()
	nextTrackChange := time.Duration(d.curTrackMeta.Duration)*time.Second - d.curPlayPos()
	d.metaLock.Unlock()
	d.state = playing
	d.setTrackChangeTimer(nextTrackChange)
	d.stopwatch.Start()
	d.InvokeOnPlaying()
	return nil
}

func (d *DLNAPlayer) Pause() error {
	if d.state != playing {
		return nil
	}

	if err := d.avTransport.Pause(context.Background()); err != nil {
		return err
	}
	d.setTrackChangeTimer(0)
	d.stopwatch.Stop()
	d.state = paused
	d.InvokeOnPaused()
	return nil
}

func (d *DLNAPlayer) Stop() error {
	switch d.state {
	case stopped:
		return nil
	case playing:
		if err := d.avTransport.Pause(context.Background()); err != nil {
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
	if d.state == paused {
		d.pendingSeek = true
		d.pendingSeekSecs = secs
	} else {
		d.seeking = true
		if err := d.avTransport.Seek(context.Background(), int(secs)); err != nil {
			d.seeking = false
			return err
		}
		d.seeking = false
	}

	d.lastStartTime = int(secs)
	d.stopwatch.Reset()

	if d.state == playing {
		d.metaLock.Lock()
		nextTrackChange := time.Duration(d.curTrackMeta.Duration)*time.Second - time.Duration(secs)*time.Second
		d.metaLock.Unlock()
		d.setTrackChangeTimer(nextTrackChange)
		d.stopwatch.Start()
	}

	d.InvokeOnSeek()
	return nil
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

	return player.Status{
		State:    state,
		TimePos:  d.curPlayPos().Seconds(),
		Duration: float64(d.curTrackMeta.Duration),
	}
}

func (d *DLNAPlayer) curPlayPos() time.Duration {
	return time.Duration(d.lastStartTime)*time.Second + d.stopwatch.Elapsed()
}

func (d *DLNAPlayer) Destroy() {
	if d.proxyServer != nil {
		go d.proxyServer.Shutdown(context.Background())
	}
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
	nextTrackChange := time.Duration(d.curTrackMeta.Duration) * time.Second
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

	// Create a new request to the target server
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

	for i := 0; i < len(d.proxyURLs); i++ {
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
	for i := 0; i < len(d.proxyURLs); i++ {
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

type retryLogger struct{}

func (retryLogger) Error(msg string, keysAndValues ...interface{}) {
	log.Println(msg, keysAndValues)
}

func (retryLogger) Info(msg string, keysAndValues ...interface{}) {
	log.Println(msg, keysAndValues)
}

func (retryLogger) Warn(msg string, keysAndValues ...interface{}) {
	log.Println(msg, keysAndValues)
}

func (retryLogger) Debug(msg string, keysAndValues ...interface{}) {
	// log only retries, not every request
	if strings.Contains(msg, "retrying request") {
		log.Println(msg, keysAndValues)
	}
}
