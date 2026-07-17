package fetch

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

// ErrResponseTooLarge is returned when a response body exceeds Config.MaxResponseBytes.
var ErrResponseTooLarge = errors.New("fetch: response exceeds maximum allowed size")

// ErrChallenge is returned when a response is classified as an access
// challenge (e.g. Cloudflare) rather than the requested page. Detection is
// deliberately narrow: an ordinary parse failure on unrelated HTML must
// never be classified as a challenge.
var ErrChallenge = errors.New("fetch: challenge required")

// challengePlatformMarker must be present for a 2xx HTML body to be
// considered for challenge classification (no cf-mitigated header case).
// It alone is not sufficient — some ordinary pages reference Cloudflare's
// challenge-platform assets for unrelated widgets — so a second, independent
// signal is required: either the interstitial's own JS bootstrap object
// (_cf_chl_opt, present across interstitial/managed/JS challenge variants,
// even when the page title isn't the fixed "Just a moment..." text observed
// in some responses) or that fixed title itself.
var (
	challengePlatformMarker = []byte("cdn-cgi/challenge-platform")
	challengeOptionsMarker  = []byte("_cf_chl_opt")
	challengeTitleMarker    = []byte("Just a moment...")
)

func looksLikeChallengeHTML(contentType string, body []byte) bool {
	if strings.TrimSpace(contentType) == "" {
		contentType = http.DetectContentType(body)
	}
	if !strings.Contains(strings.ToLower(contentType), "html") {
		return false
	}
	if !bytes.Contains(body, challengePlatformMarker) {
		return false
	}
	return bytes.Contains(body, challengeOptionsMarker) || bytes.Contains(body, challengeTitleMarker)
}

// StatusError reports a non-2xx HTTP response for a specific request URL.
type StatusError struct {
	StatusCode int
	URL        string
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("fetch: unexpected status code %d for %s", e.StatusCode, e.URL)
}

const (
	defaultMaxAttempts = 3
	defaultBaseDelay   = time.Second
	defaultMaxDelay    = 30 * time.Second
	defaultRatePerSec  = 1.0
	defaultRateBurst   = 1
)

// Config assembles everything a Client needs. The root package is
// responsible for translating its public ClientConfig into this shape and
// for supplying CheckRedirect (typically internal/siteurl.SameOriginRedirect)
// so this package never has to import the root package.
type Config struct {
	HTTPClient       *http.Client
	UserAgent        string
	MaxResponseBytes int64
	CheckRedirect    func(req *http.Request, via []*http.Request) error

	RateLimitDisabled bool
	RateLimit         float64
	RateBurst         int

	RetryDisabled    bool
	RetryMaxAttempts int
	RetryBaseDelay   time.Duration
	RetryMaxDelay    time.Duration

	// Test hooks; zero values fall back to real time/randomness/limiting.
	sleep    func(context.Context, time.Duration) error
	now      func() time.Time
	jitter   func() float64
	rateWait func(context.Context) error
}

// Client performs bounded, rate-limited, retrying HTTP GET requests. All
// requests issued through one Client share the same rate limiter.
type Client struct {
	httpClient       *http.Client
	userAgent        string
	maxResponseBytes int64
	waitLimiter      func(context.Context) error

	retryDisabled    bool
	retryMaxAttempts int
	retryBaseDelay   time.Duration
	retryMaxDelay    time.Duration

	sleep  func(context.Context, time.Duration) error
	now    func() time.Time
	jitter func() float64
}

func New(cfg Config) (*Client, error) {
	if cfg.MaxResponseBytes <= 0 {
		return nil, fmt.Errorf("fetch: MaxResponseBytes must be > 0")
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	if cfg.CheckRedirect != nil {
		cloned := *httpClient
		cloned.CheckRedirect = cfg.CheckRedirect
		httpClient = &cloned
	}

	var limiter *rate.Limiter
	if !cfg.RateLimitDisabled {
		rps := cfg.RateLimit
		if rps <= 0 {
			rps = defaultRatePerSec
		}
		burst := cfg.RateBurst
		if burst <= 0 {
			burst = defaultRateBurst
		}
		limiter = rate.NewLimiter(rate.Limit(rps), burst)
	}

	maxAttempts := cfg.RetryMaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = defaultMaxAttempts
	}
	baseDelay := cfg.RetryBaseDelay
	if baseDelay <= 0 {
		baseDelay = defaultBaseDelay
	}
	maxDelay := cfg.RetryMaxDelay
	if maxDelay <= 0 {
		maxDelay = defaultMaxDelay
	}

	sleep := cfg.sleep
	if sleep == nil {
		sleep = defaultSleep
	}
	now := cfg.now
	if now == nil {
		now = time.Now
	}
	jitter := cfg.jitter
	if jitter == nil {
		jitter = rand.Float64
	}
	waitLimiter := cfg.rateWait
	if waitLimiter == nil {
		if limiter != nil {
			waitLimiter = limiter.Wait
		} else {
			waitLimiter = func(context.Context) error { return nil }
		}
	}

	return &Client{
		httpClient:       httpClient,
		userAgent:        cfg.UserAgent,
		maxResponseBytes: cfg.MaxResponseBytes,
		waitLimiter:      waitLimiter,
		retryDisabled:    cfg.RetryDisabled,
		retryMaxAttempts: maxAttempts,
		retryBaseDelay:   baseDelay,
		retryMaxDelay:    maxDelay,
		sleep:            sleep,
		now:              now,
		jitter:           jitter,
	}, nil
}

func defaultSleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// Get performs one or more rate-limited requests, retrying on retryable
// statuses and retryable network errors per the configured policy, and
// returns the successful response body. The shared limiter is awaited before
// every attempt, including retries, so retries cannot bypass the budget.
func (c *Client) Get(ctx context.Context, u *url.URL) ([]byte, error) {
	maxAttempts := 1
	if !c.retryDisabled {
		maxAttempts = c.retryMaxAttempts
	}

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if err := c.waitLimiter(ctx); err != nil {
			return nil, fmt.Errorf("fetch: rate limiter wait: %w", err)
		}

		body, header, err := c.doAttempt(ctx, u)
		if err == nil {
			return body, nil
		}
		lastErr = err

		var statusErr *StatusError
		switch {
		case errors.As(err, &statusErr):
			if !retryableStatus(statusErr.StatusCode) {
				return nil, err
			}
		case isRetryableRequestErr(err):
			// fall through to retry
		default:
			return nil, err
		}
		if attempt == maxAttempts-1 {
			break
		}

		delay := c.nextDelay(header, attempt)
		if err := c.sleep(ctx, delay); err != nil {
			return nil, err
		}
	}
	return nil, lastErr
}

// doAttempt performs a single GET attempt. On a non-2xx status it returns the
// response headers alongside a *StatusError so the retry loop can classify
// the failure and honor Retry-After without a second round trip.
func (c *Client) doAttempt(ctx context.Context, u *url.URL) ([]byte, http.Header, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch: new request: %w", err)
	}
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch: do request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.Header.Get("cf-mitigated") == "challenge" {
		return nil, nil, ErrChallenge
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.Header, &StatusError{StatusCode: resp.StatusCode, URL: u.String()}
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, c.maxResponseBytes+1))
	if err != nil {
		return nil, nil, fmt.Errorf("fetch: read body: %w", err)
	}
	if int64(len(body)) > c.maxResponseBytes {
		return nil, nil, ErrResponseTooLarge
	}
	if looksLikeChallengeHTML(resp.Header.Get("Content-Type"), body) {
		return nil, nil, ErrChallenge
	}
	return body, nil, nil
}
