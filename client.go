package httpstream

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type Client struct {
	url string
	c   *http.Client
}

func NewClient(c *http.Client, url string) *Client {
	if c == nil {
		c = http.DefaultClient
	}
	return &Client{
		url: url,
		c:   c,
	}
}

func (c *Client) attachWriter(id string, r *io.PipeReader) {
	defer r.Close()
	resp, err := c.c.Post(c.url+"?id="+url.QueryEscape(id), "application/binary", r)
	if err != nil {
		r.CloseWithError(err)
		return
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		r.CloseWithError(fmt.Errorf("non-200 response: %s", resp.Status))
	}
}
func (c *Client) Open() (io.ReadWriteCloser, error) {
	resp, err := c.c.Post(c.url, "", nil)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("non-200 response: %s", resp.Status)
	}
	r, w := io.Pipe()
	go c.attachWriter(resp.Header.Get("Stream-ID"), r)
	return &pipe{resp.Body, w}, nil
}
func (c *Client) Accept() (io.ReadWriteCloser, error) {
	resp, err := c.c.Get(c.url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("non-200 response: %s", resp.Status)
	}
	r, w := io.Pipe()
	go c.attachWriter(resp.Header.Get("Stream-ID"), r)
	return &pipe{resp.Body, w}, nil
}
