package javdbapi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/RPbro/javdbapi/internal/fetch"
	"github.com/RPbro/javdbapi/internal/scrape"
	"github.com/RPbro/javdbapi/internal/siteurl"
)

func newDocument(body []byte) (*goquery.Document, error) {
	return goquery.NewDocumentFromReader(bytes.NewReader(body))
}

// Client is the SDK entry point. All requests issued through one Client
// share the same rate limiter, per the design's shared-budget requirement.
type Client struct {
	baseURL *url.URL
	locale  Locale
	fetcher *fetch.Client
	logger  *slog.Logger
}

func NewClient(config ClientConfig) (*Client, error) {
	cfg, err := normalizeClientConfig(config)
	if err != nil {
		return nil, err
	}

	baseURL, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid base url", ErrInvalidConfig)
	}
	if baseURL.Host == "" {
		return nil, fmt.Errorf("%w: invalid base url: missing host", ErrInvalidConfig)
	}
	if baseURL.User != nil {
		return nil, fmt.Errorf("%w: base url must not contain user info", ErrInvalidConfig)
	}
	if baseURL.Scheme != "http" && baseURL.Scheme != "https" {
		return nil, fmt.Errorf("%w: unsupported base url scheme %q", ErrInvalidConfig, baseURL.Scheme)
	}

	httpClient, err := buildHTTPClient(cfg.HTTP)
	if err != nil {
		return nil, err
	}

	fetcher, err := fetch.New(fetch.Config{
		HTTPClient:        httpClient,
		UserAgent:         cfg.UserAgent,
		MaxResponseBytes:  cfg.HTTP.MaxResponseBytes,
		CheckRedirect:     checkSiteRedirect,
		RateLimitDisabled: cfg.RateLimit.Disabled,
		RateLimit:         cfg.RateLimit.RequestsPerSecond,
		RateBurst:         cfg.RateLimit.Burst,
		RetryDisabled:     cfg.Retry.Disabled,
		RetryMaxAttempts:  cfg.Retry.MaxAttempts,
		RetryBaseDelay:    cfg.Retry.BaseDelay,
		RetryMaxDelay:     cfg.Retry.MaxDelay,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}

	return &Client{
		baseURL: baseURL,
		locale:  cfg.Locale,
		fetcher: fetcher,
		logger:  cfg.Logger,
	}, nil
}

// buildHTTPClient constructs a cloned default transport with an optional
// proxy. The caller's own HTTP.Client, if set, is used and never mutated.
func buildHTTPClient(cfg HTTPConfig) (*http.Client, error) {
	if cfg.Client != nil {
		return cfg.Client, nil
	}

	baseTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, fmt.Errorf("%w: default transport is not *http.Transport", ErrInvalidConfig)
	}
	transport := baseTransport.Clone()

	if proxyURL := strings.TrimSpace(cfg.ProxyURL); proxyURL != "" {
		parsed, err := url.Parse(proxyURL)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid proxy url", ErrInvalidConfig)
		}
		scheme := strings.ToLower(parsed.Scheme)
		switch scheme {
		case "http", "https", "socks5", "socks5h":
		default:
			return nil, fmt.Errorf("%w: unsupported proxy scheme %q", ErrInvalidConfig, scheme)
		}
		parsed.Scheme = scheme
		if parsed.Host == "" {
			return nil, fmt.Errorf("%w: invalid proxy url: missing host", ErrInvalidConfig)
		}
		transport.Proxy = http.ProxyURL(parsed)
	}

	return &http.Client{Timeout: cfg.Timeout, Transport: transport}, nil
}

// VideoURL returns a human-shareable link for id. id is expected to come
// from ParseVideoID or a prior VideoSummary.ID, so URL construction cannot
// fail in practice; on the defensive error path (a hand-built invalid ID)
// it returns an empty string rather than a malformed link.
func (c *Client) VideoURL(id VideoID) string {
	target, err := siteurl.Video(c.baseURL, string(id), string(c.locale))
	if err != nil {
		return ""
	}
	return target.String()
}

// sanitizedURL strips query values (which may contain the caller's search
// keyword or other sensitive filters) before a URL is surfaced in an error
// or log line.
func sanitizedURL(u *url.URL) string {
	if u == nil {
		return ""
	}
	clone := *u
	clone.User = nil
	clone.RawQuery = ""
	clone.ForceQuery = false
	clone.Fragment = ""
	clone.RawFragment = ""
	return clone.String()
}

func mapFetchErr(op string, target *url.URL, err error) error {
	var statusErr *fetch.StatusError
	switch {
	case errors.Is(err, fetch.ErrChallenge):
		return &OpError{Op: op, URL: sanitizedURL(target), Err: ErrChallenge}
	case errors.Is(err, ErrAuthenticationRequired):
		return &OpError{Op: op, URL: sanitizedURL(target), Err: ErrAuthenticationRequired}
	case errors.As(err, &statusErr):
		return &OpError{Op: op, URL: sanitizedURL(target), Err: &HTTPError{StatusCode: statusErr.StatusCode, URL: sanitizedURL(target)}}
	default:
		return &OpError{Op: op, URL: sanitizedURL(target), Err: err}
	}
}

func mapScrapeErr(op string, target *url.URL, err error) error {
	switch {
	case errors.Is(err, scrape.ErrEmpty):
		return &OpError{Op: op, URL: sanitizedURL(target), Err: ErrEmptyResult}
	case errors.Is(err, scrape.ErrParse):
		return &OpError{Op: op, URL: sanitizedURL(target), Err: fmt.Errorf("%w: %v", ErrParse, err)}
	default:
		return &OpError{Op: op, URL: sanitizedURL(target), Err: err}
	}
}

func checkSiteRedirect(req *http.Request, via []*http.Request) error {
	if err := siteurl.SameOriginRedirect(req.URL, via); err != nil {
		return err
	}
	if req.URL != nil && req.URL.Path == "/login" {
		return ErrAuthenticationRequired
	}
	return nil
}

func (c *Client) logWarnings(op string, warnings []scrape.Warning) {
	if c.logger == nil {
		return
	}
	for _, w := range warnings {
		c.logger.Warn("parse warning", "op", op, "field", w.Field, "message", w.Message)
	}
}

// toPublicScore copies a scrape-layer score into a new public Score so the
// internal and public models never share a mutable pointer.
func toPublicScore(s *scrape.Score) *Score {
	if s == nil {
		return nil
	}
	return &Score{Value: s.Value, Count: s.Count}
}

func toPublicSummary(s scrape.Summary) VideoSummary {
	return VideoSummary{
		ID:          VideoID(s.ID),
		Code:        s.Code,
		Title:       s.Title,
		CoverURL:    s.CoverURL,
		PublishedAt: s.PublishedAt,
		Score:       toPublicScore(s.Score),
		Availability: Availability{
			HasMagnet:         s.Availability.HasMagnet,
			HasSubtitleMagnet: s.Availability.HasSubtitleMagnet,
			IsPlayable:        s.Availability.IsPlayable,
		},
	}
}

func toPublicListPage(p scrape.ListPage) Page[VideoSummary] {
	items := make([]VideoSummary, len(p.Items))
	for i, s := range p.Items {
		items[i] = toPublicSummary(s)
	}
	return NewPage(p.Number, items, p.HasNext)
}

// fetchList runs the shared fetch -> parse-document -> ParseList -> map flow
// used by every list endpoint (Home, Search, MakerVideos, ActorVideos, Ranking).
func (c *Client) fetchList(ctx context.Context, op string, target *url.URL, page int) (Page[VideoSummary], error) {
	body, err := c.fetcher.Get(ctx, target)
	if err != nil {
		return Page[VideoSummary]{}, mapFetchErr(op+".fetch", target, err)
	}

	doc, err := newDocument(body)
	if err != nil {
		return Page[VideoSummary]{}, &OpError{Op: op + ".document", URL: sanitizedURL(target), Err: fmt.Errorf("%w: %v", ErrParse, err)}
	}

	parsed, err := scrape.ParseList(doc, page)
	if err != nil {
		return Page[VideoSummary]{}, mapScrapeErr(op+".parse", target, err)
	}
	c.logWarnings(op+".parse", parsed.Warnings)
	return toPublicListPage(parsed.Value), nil
}
