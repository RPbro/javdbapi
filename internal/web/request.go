package web

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Runner struct {
	client    *http.Client
	userAgent string
	debug     bool
}

type UnexpectedStatusError struct {
	StatusCode int
	URL        string
}

func (e *UnexpectedStatusError) Error() string {
	if e.URL == "" {
		return fmt.Sprintf("unexpected status code: %d", e.StatusCode)
	}
	return fmt.Sprintf("unexpected status code: %d url=%s", e.StatusCode, e.URL)
}

func NewRunner(timeout time.Duration, proxyURL string, userAgent string, debug bool) (*Runner, error) {
	baseTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, fmt.Errorf("default transport is not *http.Transport")
	}

	transport := baseTransport.Clone()

	proxyURL = strings.TrimSpace(proxyURL)
	if proxyURL != "" {
		parsed, err := url.Parse(proxyURL)
		if err != nil {
			return nil, fmt.Errorf("parse proxy url: %w", err)
		}

		scheme := strings.ToLower(parsed.Scheme)
		switch scheme {
		case "http", "https", "socks5", "socks5h":
		default:
			return nil, fmt.Errorf("unsupported proxy scheme: %q", parsed.Scheme)
		}
		parsed.Scheme = scheme

		if parsed.Scheme == "" || parsed.Host == "" || parsed.Hostname() == "" {
			return nil, fmt.Errorf("invalid proxy url: %q", proxyURL)
		}
		transport.Proxy = http.ProxyURL(parsed)
	}

	client := &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}

	return &Runner{
		client:    client,
		userAgent: userAgent,
		debug:     debug,
	}, nil
}

func (r *Runner) Get(ctx context.Context, rawURL string) ([]byte, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("runner is not initialized")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	if r.userAgent != "" {
		req.Header.Set("User-Agent", r.userAgent)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, &UnexpectedStatusError{StatusCode: resp.StatusCode, URL: rawURL}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	return body, nil
}
