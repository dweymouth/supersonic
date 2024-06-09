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
	conn, err := Dial()
	if err != nil {
		return nil, err
	}
	client := &Client{httpC: http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return conn, nil
			},
		},
	}}
	if err := client.Ping(); err != nil {
		return nil, err
	}
	return client, nil
}

func (c *Client) Ping() error {
	if c.makeSimpleRequest(http.MethodGet, PingPath) != nil {
		return ErrPingFail
	}
	return nil
}

func (c *Client) Play() error {
	return c.makeSimpleRequest(http.MethodPost, PlayPath)
}

func (c *Client) Pause() error {
	return c.makeSimpleRequest(http.MethodPost, PausePath)
}

func (c *Client) PlayPause() error {
	return c.makeSimpleRequest(http.MethodPost, PlayPausePath)
}

func (c *Client) SeekNext() error {
	return c.makeSimpleRequest(http.MethodPost, NextPath)
}

func (c *Client) SeekBackOrPrevious() error {
	return c.makeSimpleRequest(http.MethodPost, NextPath)
}

func (c *Client) makeSimpleRequest(method string, path string) error {
	var resp *http.Response
	var err error
	switch method {
	case http.MethodGet:
		resp, err = c.httpC.Get(path)
	case http.MethodPost:
		resp, err = c.httpC.Post(path, "application/json", nil)
	}

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
