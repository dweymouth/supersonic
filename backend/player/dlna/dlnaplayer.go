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

	curTrackMeta  mediaprovider.MediaItemMetadata
	nextTrackMeta mediaprovider.MediaItemMetadata

	lastSeekSecs int
	seekedAt     time.Time

	proxyServer *http.Server
	proxyActive atomic.Bool
	localIP     string
	proxyPort   int

	proxyURLLock sync.Mutex
	proxyURLs    map[string]string
}

func NewDLNAPlayer(device *device.MediaRenderer) (*DLNAPlayer, error) {
	avt, err := device.AVTransportClient()
	if err != nil {
		return nil, err
	}
	rc, err := device.RenderingControlClient()
	if err != nil {
		return nil, err
	}
	return &DLNAPlayer{
		avTransport:   avt,
		renderControl: rc,
		proxyURLs:     make(map[string]string),
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

	d.curTrackMeta = meta
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
	d.seekedAt = time.Now()
	d.lastSeekSecs = 0
	d.InvokeOnPlaying()
	d.InvokeOnTrackChange()
	return nil
}

func (d *DLNAPlayer) SetNextFile(url string, meta mediaprovider.MediaItemMetadata) error {
	var media *avtransport.MediaItem
	d.nextTrackMeta = meta
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

func (d *DLNAPlayer) addURLToProxy(url string) string {
	hash := md5.Sum([]byte(url))
	key := base64.StdEncoding.EncodeToString(hash[:])
	d.proxyURLLock.Lock()
	d.proxyURLs[key] = url
	d.proxyURLLock.Unlock()
	return key
}

func (d *DLNAPlayer) urlForItem(key string) string {
	return fmt.Sprintf("http://%s:%d/%s", d.localIP, d.proxyPort, key)
}

func (d *DLNAPlayer) Continue() error {
	if err := d.avTransport.Play(context.Background()); err != nil {
		return err
	}
	d.state = playing
	d.InvokeOnPlaying()
	return nil
}

func (d *DLNAPlayer) Pause() error {
	if err := d.avTransport.Pause(context.Background()); err != nil {
		return err
	}
	d.state = paused
	d.InvokeOnPaused()
	return nil
}

func (d *DLNAPlayer) Stop() error {
	if err := d.avTransport.Pause(context.Background()); err != nil {
		return err
	}
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
	d.seekedAt = time.Now()
	d.lastSeekSecs = int(secs)
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

	time := time.Now().Sub(d.seekedAt) + time.Duration(d.lastSeekSecs)*time.Second

	return player.Status{
		State:    state,
		TimePos:  time.Seconds(),
		Duration: float64(d.curTrackMeta.Duration),
	}
}

func getLocalIP() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}

		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}

	return "", fmt.Errorf("no suitable interface found")
}

func (d *DLNAPlayer) ensureSetupProxy() error {
	if d.proxyActive.Swap(true) {
		return nil // already active
	}

	var err error
	d.localIP, err = getLocalIP()
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

type proxy struct {
	url string
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
