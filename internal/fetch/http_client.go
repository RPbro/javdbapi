package fetch

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// NewHTTPClient clones the default transport and optionally overrides its proxy.
func NewHTTPClient(timeout time.Duration, proxyURL string) (*http.Client, error) {
	baseTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, errors.New("default transport is not *http.Transport")
	}
	transport := baseTransport.Clone()

	if proxyURL := strings.TrimSpace(proxyURL); proxyURL != "" {
		parsed, err := url.Parse(proxyURL)
		if err != nil {
			return nil, errors.New("invalid proxy url")
		}
		scheme := strings.ToLower(parsed.Scheme)
		switch scheme {
		case "http", "https", "socks5", "socks5h":
		default:
			return nil, fmt.Errorf("unsupported proxy scheme %q", scheme)
		}
		parsed.Scheme = scheme
		if parsed.Host == "" {
			return nil, errors.New("invalid proxy url: missing host")
		}
		transport.Proxy = http.ProxyURL(parsed)
	}

	return &http.Client{Timeout: timeout, Transport: transport}, nil
}
