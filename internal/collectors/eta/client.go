package eta

import (
	"context"
	"io"
	"net/http"
)

type Client struct {
	BaseURL string
	Client  *http.Client
}

func NewClient(url string) *Client {
	return &Client{
		BaseURL: url,
		Client:  &http.Client{},
	}
}

func (c *Client) Fetch(ctx context.Context) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}
