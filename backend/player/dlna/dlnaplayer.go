package dlna

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"sync/atomic"

	"github.com/dweymouth/supersonic/backend/player"
	"github.com/supersonic-app/go-upnpcast/device"
	"github.com/supersonic-app/go-upnpcast/services/avtransport"
)

const (
	stopped = 0
	playing = 1
	paused  = 2
)

var unimplemented = errors.New("unimplemented")

type DLNAPlayer struct {
	player.BasePlayerCallbackImpl

	avTransport *avtransport.Client

	state   int // stopped, playing, paused
	seeking bool
}

func NewDLNAPlayer(device *device.MediaRenderer) (*DLNAPlayer, error) {
	avt, err := device.AVTransportClient()
	if err != nil {
		return nil, err
	}
	return &DLNAPlayer{avTransport: avt}, nil
}

func (d *DLNAPlayer) SetVolume(vol int) error {
	return unimplemented
}

func (d *DLNAPlayer) GetVolume() int {
	return 0
}

func (d *DLNAPlayer) PlayFile(urlstr string) error {
	ensureSetupProxies()

	proxyURLLock.Lock()
	dlnaProxyCurrent.url = urlstr
	proxyURLLock.Unlock()

	media := avtransport.MediaItem{
		URL:   "http://" + localIP + ":8080/current",
		Title: "Supersonic media item",
	}
	log.Printf("URL %s", media.URL)

	err := d.avTransport.SetAVTransportMedia(context.Background(), &media)
	if err != nil {
		return err
	}
	if err := d.avTransport.Play(context.Background()); err != nil {
		return err
	}
	d.state = playing
	d.InvokeOnPlaying()
	return nil
}

func (d *DLNAPlayer) SetNextFile(url string) error {
	var media *avtransport.MediaItem
	if url != "" {
		ensureSetupProxies()

		proxyURLLock.Lock()
		dlnaProxyCurrent.url = url
		proxyURLLock.Unlock()

		media = &avtransport.MediaItem{
			URL: "http://" + localIP + ":8080/next",
		}
	}
	return d.avTransport.SetNextAVTransportMedia(context.Background(), media)
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

	// TODO - the rest

	return player.Status{
		State: state,
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

var (
	localIP          string
	proxyURLLock     sync.Mutex
	dlnaProxyCurrent proxy
	dlnaProxyNext    proxy
	proxyActive      atomic.Bool
)

func ensureSetupProxies() {
	if proxyActive.Swap(true) {
		return // already active
	}

	localIP, _ = getLocalIP()
	log.Println(localIP)

	mux := http.NewServeMux()
	mux.HandleFunc("/current", dlnaProxyCurrent.handleRequest)
	mux.HandleFunc("/next", dlnaProxyNext.handleRequest)
	go http.ListenAndServe(":8080", mux)
}

type proxy struct {
	url string
}

func (p *proxy) handleRequest(w http.ResponseWriter, r *http.Request) {
	// Create a new request to the target server
	proxyURLLock.Lock()
	url := p.url
	proxyURLLock.Unlock()
	log.Println("Got request for " + url)
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
