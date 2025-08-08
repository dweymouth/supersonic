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
	if c.sendRequest(PingPath) != nil {
		return ErrPingFail
	}
	return nil
}

func (c *Client) Play() error {
	return c.sendRequest(PlayPath)
}

func (c *Client) Pause() error {
	return c.sendRequest(PausePath)
}

func (c *Client) PlayPause() error {
	return c.sendRequest(PlayPausePath)
}

func (c *Client) Stop() error {
	return c.sendRequest(StopPath)
}

func (c *Client) StopAfterCurrent() error {
	return c.sendRequest(StopAfterCurrentPath)
}

func (c *Client) SeekNext() error {
	return c.sendRequest(NextPath)
}

func (c *Client) SeekBackOrPrevious() error {
	return c.sendRequest(PreviousPath)
}

func (c *Client) SeekSeconds(secs float64) error {
	return c.sendRequest(SeekToSecondsPath(secs))
}

func (c *Client) SeekBySeconds(secs float64) error {
	return c.sendRequest(SeekBySecondsPath(secs))
}

func (c *Client) SetVolume(vol int) error {
	return c.sendRequest(SetVolumePath(vol))
}

func (c *Client) AdjustVolumePct(pct float64) error {
	return c.sendRequest(AdjustVolumePctPath(pct))
}

func (c *Client) Show() error {
	return c.sendRequest(ShowPath)
}

func (c *Client) Quit() error {
	return c.sendRequest(QuitPath)
}

func (c *Client) sendRequest(path string) error {
	resp, err := c.httpC.Get("http://supersonic/" + path)

	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var r Response
		json.NewDecoder(resp.Body).Decode(&r)
		return errors.New(r.Error)
	}
	return nil
}
