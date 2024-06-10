package ipc

import (
	"context"
	"encoding/json"
	"errors"
	"log"
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
		log.Println("dial error")
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
		log.Println("ping error")
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

func (c *Client) Show() error {
	return c.makeSimpleRequest(http.MethodPost, ShowPath)
}

func (c *Client) Quit() error {
	return c.makeSimpleRequest(http.MethodPost, QuitPath)
}

func (c *Client) makeSimpleRequest(method string, path string) error {
	var resp *http.Response
	var err error
	switch method {
	case http.MethodGet:
		resp, err = c.httpC.Get("http://supersonic/" + path)
	case http.MethodPost:
		resp, err = c.httpC.Post("http://supersonic/"+path, "application/json", nil)
	}

	if err != nil {
		log.Printf("http err: %v\n", err)
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
