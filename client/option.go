package client

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/forbearing/gst/types"
	"golang.org/x/time/rate"
)

type Option[M, REQ, RSP any] func(*Client[M, REQ, RSP])

func WithContext[M, REQ, RSP any](ctx context.Context) Option[M, REQ, RSP] {
	return func(c *Client[M, REQ, RSP]) {
		if ctx != nil {
			c.ctx = ctx
		}
	}
}

func WithHTTPClient[M, REQ, RSP any](client *http.Client) Option[M, REQ, RSP] {
	return func(c *Client[M, REQ, RSP]) {
		if client != nil {
			c.httpClient = client
		}
	}
}

func WithHeader[M, REQ, RSP any](header http.Header) Option[M, REQ, RSP] {
	return func(c *Client[M, REQ, RSP]) {
		if header != nil {
			c.header = header.Clone()
		}
	}
}

func WithDebug[M, REQ, RSP any]() Option[M, REQ, RSP] {
	return func(c *Client[M, REQ, RSP]) {
		c.debug = true
	}
}

func WithRetry[M, REQ, RSP any](maxRetries int, wait time.Duration) Option[M, REQ, RSP] {
	return func(c *Client[M, REQ, RSP]) {
		if maxRetries < 0 {
			maxRetries = 0
		}
		if wait < 0 {
			wait = 0
		}
		c.maxRetries = maxRetries
		c.retryWait = wait
	}
}

func WithRateLimit[M, REQ, RSP any](r rate.Limit, b int) Option[M, REQ, RSP] {
	return func(c *Client[M, REQ, RSP]) {
		if r <= 0 || b <= 0 {
			return
		}
		c.rateLimiter = rate.NewLimiter(r, b)
	}
}

func WithLogger[M, REQ, RSP any](logger types.Logger) Option[M, REQ, RSP] {
	return func(c *Client[M, REQ, RSP]) {
		if logger != nil {
			c.Logger = logger
		}
	}
}

func WithTimeout[M, REQ, RSP any](timeout time.Duration) Option[M, REQ, RSP] {
	return func(c *Client[M, REQ, RSP]) {
		if timeout <= 0 {
			return
		}
		if c.httpClient == nil {
			c.httpClient = http.DefaultClient
		}
		c.httpClient.Timeout = timeout
	}
}

func WithUserAgent[M, REQ, RSP any](userAgent string) Option[M, REQ, RSP] {
	return func(c *Client[M, REQ, RSP]) {
		if c.header == nil {
			c.header = http.Header{}
		}
		c.header.Set("User-Agent", userAgent)
	}
}

func WithBaseAuth[M, REQ, RSP any](username, password string) Option[M, REQ, RSP] {
	return func(c *Client[M, REQ, RSP]) {
		if username = strings.TrimSpace(username); len(username) != 0 {
			c.username = username
			c.password = password
		}
	}
}

func WithToken[M, REQ, RSP any](token string) Option[M, REQ, RSP] {
	return func(c *Client[M, REQ, RSP]) {
		if token = strings.TrimSpace(token); len(token) != 0 {
			c.token = token
		}
	}
}
