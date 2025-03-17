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

	proxyURLLock sync.Mutex
	proxyURLs    map[string]string

	timerActive atomic.Bool
	timer       *time.Timer
	resetChan   chan (time.Duration)
}

func NewDLNAPlayer(device *device.MediaRenderer) (*DLNAPlayer, error) {
	retry := retryablehttp.NewClient()
	retry.RetryMax = 3
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
		proxyURLs:     make(map[string]string),
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

	log.Println("playing track " + meta.Name)

	d.metaLock.Lock()
	d.curTrackMeta = meta
	d.metaLock.Unlock()
	key := d.addURLToProxy(urlstr)

	media := avtransport.MediaItem{
		URL:   d.urlForItem(key),
		Title: meta.Name,
	}

	err := d.avTransport.SetAVTransportMedia(context.Background(), &media)
	if err != nil {
		return err
	}
	if err := d.avTransport.Play(context.Background()); err != nil {
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
	return d.avTransport.SetNextAVTransportMedia(context.Background(), media)
}

func (d *DLNAPlayer) Continue() error {
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
	if err := d.avTransport.Pause(context.Background()); err != nil {
		return err
	}
	d.setTrackChangeTimer(0)
	d.stopwatch.Reset()
	d.lastStartTime = 0
	d.state = stopped
	d.InvokeOnStopped()
	return nil
}

func (d *DLNAPlayer) SeekSeconds(secs float64) error {
	d.seeking = true
	if err := d.avTransport.Seek(context.Background(), int(secs)); err != nil {
		d.seeking = false
		return err
	}
	d.seeking = false

	d.metaLock.Lock()
	nextTrackChange := time.Duration(d.curTrackMeta.Duration)*time.Second - time.Duration(secs)*time.Second
	d.metaLock.Unlock()
	d.setTrackChangeTimer(nextTrackChange)
	d.lastStartTime = int(secs)
	d.stopwatch.Reset()
	if d.state == playing {
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

func (d *DLNAPlayer) addURLToProxy(url string) string {
	hash := md5.Sum([]byte(url))
	key := base64.StdEncoding.EncodeToString(hash[:])
	d.proxyURLLock.Lock()
	d.proxyURLs[key] = url
	d.proxyURLLock.Unlock()
	return key
}

func (d *DLNAPlayer) setTrackChangeTimer(dur time.Duration) {
	if d.timerActive.Swap(true) {
		// was active
		log.Println("timer was active")
		d.resetChan <- dur
		log.Println("and reset")
		return
	}
	if dur == 0 {
		d.timerActive.Store(false)
		return
	}
	log.Println("starting timer")

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
	var url string
	key := strings.TrimPrefix(r.URL.Path, "/")
	d.proxyURLLock.Lock()
	if u, ok := d.proxyURLs[key]; ok {
		url = u
	}
	d.proxyURLLock.Unlock()

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
