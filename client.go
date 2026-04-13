package javdbapi

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/RPbro/javdbapi/internal/web"
)

const (
	defaultBaseURL   = "https://javdb.com"
	defaultTimeout   = 30 * time.Second
	defaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36"
)

type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
	proxyURL   string
	userAgent  string
	debug      bool
	runner     *web.Runner
}

func NewClient(cfg Config) (*Client, error) {
	baseURLStr := strings.TrimSpace(cfg.BaseURL)
	if baseURLStr == "" {
		baseURLStr = defaultBaseURL
	}

	baseURL, err := url.Parse(baseURLStr)
	if err != nil || baseURL.Scheme == "" || baseURL.Host == "" {
		return nil, fmt.Errorf("%w: invalid base url %q", ErrInvalidConfig, baseURLStr)
	}
	if baseURL.Scheme != "http" && baseURL.Scheme != "https" {
		return nil, fmt.Errorf("%w: invalid base url %q", ErrInvalidConfig, baseURLStr)
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	userAgent := strings.TrimSpace(cfg.UserAgent)
	if userAgent == "" {
		userAgent = defaultUserAgent
	}

	httpClient := &http.Client{Timeout: timeout}

	proxyURL := strings.TrimSpace(cfg.ProxyURL)

	runner, err := web.NewRunner(timeout, cfg.ProxyURL, userAgent, cfg.Debug)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidConfig, err)
	}

	return &Client{
		baseURL:    baseURL,
		httpClient: httpClient,
		proxyURL:   proxyURL,
		userAgent:  userAgent,
		debug:      cfg.Debug,
		runner:     runner,
	}, nil
}
