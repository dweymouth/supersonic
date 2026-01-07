package ipc

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
)

var ErrPingFail = errors.New("ping failed")

type Client struct {
	httpC http.Client
}

// Connect attempts to connect to the IPC socket as client.
func Connect() (*Client, error) {
	client := &Client{httpC: http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return Dial()
			},
		},
	}}
	if err := client.Ping(); err != nil {
		return nil, err
	}
	return client, nil
}

func (c *Client) Ping() error {
	if _, err := c.sendRequest(PingPath); err != nil {
		return ErrPingFail
	}
	return nil
}

func (c *Client) Play() error {
	_, err := c.sendRequest(PlayPath)
	return err
}

func (c *Client) Pause() error {
	_, err := c.sendRequest(PausePath)
	return err
}

func (c *Client) PlayAlbum(id string, firstTrack int, shuffle bool) error {
	_, err := c.sendRequest(BuildPlayAlbumPath(id, firstTrack, shuffle))
	return err
}

func (c *Client) PlayPlaylist(id string, firstTrack int, shuffle bool) error {
	_, err := c.sendRequest(BuildPlayPlaylistPath(id, firstTrack, shuffle))
	return err
}

func (c *Client) SearchAlbum(search string) (string, error) {
	return c.sendRequest(BuildSearchAlbumPath(search))
}

func (c *Client) SearchPlaylist(search string) (string, error) {
	return c.sendRequest(BuildSearchPlaylistPath(search))
}

func (c *Client) SearchTrack(search string) (string, error) {
	return c.sendRequest(BuildSearchTrackPath(search))
}

func (c *Client) PlayTrack(id string) error {
	_, err := c.sendRequest(BuildPlayTrackPath(id))
	return err
}

func (c *Client) PlayPause() error {
	_, err := c.sendRequest(PlayPausePath)
	return err
}

func (c *Client) Stop() error {
	_, err := c.sendRequest(StopPath)
	return err
}

func (c *Client) PauseAfterCurrent() error {
	_, err := c.sendRequest(PauseAfterCurrentPath)
	return err
}

func (c *Client) SeekNext() error {
	_, err := c.sendRequest(NextPath)
	return err
}

func (c *Client) SeekBackOrPrevious() error {
	_, err := c.sendRequest(PreviousPath)
	return err
}

func (c *Client) SeekSeconds(secs float64) error {
	_, err := c.sendRequest(SeekToSecondsPath(secs))
	return err
}

func (c *Client) SeekBySeconds(secs float64) error {
	_, err := c.sendRequest(SeekBySecondsPath(secs))
	return err
}

func (c *Client) SetVolume(vol int) error {
	_, err := c.sendRequest(SetVolumePath(vol))
	return err
}

func (c *Client) AdjustVolumePct(pct float64) error {
	_, err := c.sendRequest(AdjustVolumePctPath(pct))
	return err
}

func (c *Client) Show() error {
	_, err := c.sendRequest(ShowPath)
	return err
}

func (c *Client) Quit() error {
	_, err := c.sendRequest(QuitPath)
	return err
}

func (c *Client) sendRequest(path string) (string, error) {
	resp, err := c.httpC.Get("http://supersonic/" + path)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var r Response
	json.NewDecoder(resp.Body).Decode(&r)
	if resp.StatusCode != http.StatusOK {
		return "", errors.New(r.Error)
	}
	return string(r.Data), nil
}
