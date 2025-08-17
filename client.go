package javdbapi

import (
	"time"
)

type Client struct {
	domain  string
	ua      string
	timeout time.Duration
	proxy   string
}

type option func(c *Client)

func WithDomain(domain string) func(c *Client) {
	return func(c *Client) {
		c.domain = domain
	}
}

func WithUserAgent(ua string) func(c *Client) {
	return func(c *Client) {
		c.ua = ua
	}
}

func WithTimeout(timeout time.Duration) func(c *Client) {
	return func(c *Client) {
		c.timeout = timeout
	}
}

func WithProxy(addr string) func(c *Client) {
	return func(c *Client) {
		c.proxy = addr
	}
}

func NewClient(options ...option) *Client {
	client := &Client{
		domain:  defaultDomain,
		ua:      defaultUserAgent,
		timeout: defaultTimeout,
	}
	for _, fn := range options {
		fn(client)
	}
	return client
}
