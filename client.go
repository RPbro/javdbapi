package javdbapi

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"time"
)

const (
	defaultDomain    = "https://javdb.com"
	defaultTimeout   = time.Second * 30
	defaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.0.0 Safari/537.36"

	defaultPage    = 1
	defaultPageMax = 60

	PathReviews  = "/reviews/lastest"
	PathRankings = "/rankings/movies"
	PathMakers   = "/makers"
	PathActors   = "/actors"
)

type Client struct {
	Domain    string
	UserAgent string
	ProxyAddr string
	HTTP      *http.Client
}

type option func(c *Client)

func WithDomain(domain string) func(c *Client) {
	return func(c *Client) {
		c.Domain = domain
	}
}

func WithUserAgent(ua string) func(c *Client) {
	return func(c *Client) {
		c.UserAgent = ua
	}
}

func WithProxy(addr string) func(c *Client) {
	return func(c *Client) {
		c.ProxyAddr = addr
	}
}

func WithTimeout(timeout time.Duration) func(c *Client) {
	return func(c *Client) {
		c.HTTP.Timeout = timeout
	}
}

func NewClient(options ...option) *Client {
	client := &Client{
		Domain:    defaultDomain,
		UserAgent: defaultUserAgent,
		HTTP: &http.Client{
			Timeout: defaultTimeout,
		},
	}

	for _, fn := range options {
		fn(client)
	}

	if len(client.ProxyAddr) > 0 {
		proxyURL, _ := url.Parse(client.ProxyAddr)
		tr := &http.Transport{
			Proxy:           http.ProxyURL(proxyURL),
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client.HTTP.Transport = tr
	}

	return client
}

func (c *Client) SetClient(client *http.Client) *Client {
	c.HTTP = client
	return c
}
