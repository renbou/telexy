package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"sync"
)

// Local pool for JSON request marshalling
var bufPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

func getBuffer() *bytes.Buffer {
	return bufPool.Get().(*bytes.Buffer)
}

func putBuffer(b *bytes.Buffer) {
	b.Reset()
	bufPool.Put(b)
}

// Client is a neat wrapper for communicating with the Telegram Bot API,
// providing request and response serialization as well as authorization.
type Client struct {
	endpointURL    *url.URL
	hc             *http.Client
	methodURLCache sync.Map
}

type ClientOpts struct {
	Client *http.Client
}

func NewClient(endpoint, token string, opts *ClientOpts) (client *Client, err error) {
	client = new(Client)
	if client.endpointURL, err = url.Parse(endpoint); err != nil {
		return nil, fmt.Errorf("invalid API url: %w", err)
	}
	client.endpointURL.Path = path.Join(client.endpointURL.Path, "bot"+token)

	if opts != nil && opts.Client != nil {
		client.hc = opts.Client
	} else {
		client.hc = http.DefaultClient
	}
	return client, nil
}

func (c *Client) methodURL(method string) string {
	if v, ok := c.methodURLCache.Load(method); ok {
		return v.(string)
	}

	urlCopy := *c.endpointURL
	urlCopy.Path = path.Join(urlCopy.Path, method)
	v, _ := c.methodURLCache.LoadOrStore(method, urlCopy.String())
	return v.(string)
}

// Do makes a POST request to the API with the specified request and calls the
// consumer with a decoder ready to read the "result" field of the response.
func (c *Client) Do(ctx context.Context,
	method string, v any, consumer func(Decoder) error,
) error {
	buffer := getBuffer()
	defer putBuffer(buffer)

	encoder := json.NewEncoder(buffer)
	if err := encoder.Encode(v); err != nil {
		return fmt.Errorf("encoding %s request: %w", method, err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.methodURL(method), buffer)
	if err != nil {
		return fmt.Errorf("preparing http %s request: %w", method, err)
	}

	resp, err := c.hc.Do(req)
	if err != nil {
		// Unwrap url.Error returned from do to avoid leaking url with bot token
		return fmt.Errorf("executing http %s request: %w", method, errors.Unwrap(err))
	} else if resp.StatusCode != http.StatusOK {
		// TODO: properly handle errors as specified in https://core.telegram.org/api/errors
		return fmt.Errorf("bad api response code: %s", resp.Status)
	}

	return readResponse(resp.Body, consumer)
}

// GetUpdates completes a getUpdates request using the specified options and parses
// each of the returned updates using the specified consumer.
func (c *Client) GetUpdates(ctx context.Context,
	req GetUpdatesRequest, consumer func(UpdateInfo, Decoder) error,
) error {
	return c.Do(ctx, "getUpdates", req, getUpdatesResponseConsumer(consumer))
}
