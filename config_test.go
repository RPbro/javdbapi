package javdbapi

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitizedURLStripsSensitiveParts(t *testing.T) {
	u, err := url.Parse("https://user:pass@javdb.com/search?q=secret#frag")
	require.NoError(t, err)

	got := sanitizedURL(u)
	assert.Equal(t, "https://javdb.com/search", got)
	assert.NotContains(t, got, "user")
	assert.NotContains(t, got, "pass")
	assert.NotContains(t, got, "secret")
	assert.NotContains(t, got, "frag")
}

func TestClientConfigDefaults(t *testing.T) {
	got, err := normalizeClientConfig(ClientConfig{})
	require.NoError(t, err)
	assert.Equal(t, LocaleZH, got.Locale)
	assert.Equal(t, 30*time.Second, got.HTTP.Timeout)
	assert.EqualValues(t, 8<<20, got.HTTP.MaxResponseBytes)
	assert.Equal(t, 3, got.Retry.MaxAttempts)
	assert.Equal(t, 1.0, got.RateLimit.RequestsPerSecond)
	assert.Equal(t, 1, got.RateLimit.Burst)
}

func TestClientConfigRejectsCustomClientConflicts(t *testing.T) {
	_, err := normalizeClientConfig(ClientConfig{
		HTTP: HTTPConfig{Client: http.DefaultClient, Timeout: time.Second},
	})
	require.ErrorIs(t, err, ErrInvalidConfig)
}

func TestClientConfigRejectsUnsupportedLocale(t *testing.T) {
	_, err := normalizeClientConfig(ClientConfig{Locale: "en"})
	require.ErrorIs(t, err, ErrInvalidConfig)
}

func TestClientConfigRejectsInvalidEnabledValues(t *testing.T) {
	for _, cfg := range []ClientConfig{
		{Retry: RetryPolicy{MaxAttempts: -1}},
		{RateLimit: RateLimitPolicy{RequestsPerSecond: -1}},
		{RateLimit: RateLimitPolicy{Burst: -1}},
		{HTTP: HTTPConfig{MaxResponseBytes: -1}},
	} {
		_, err := normalizeClientConfig(cfg)
		require.ErrorIs(t, err, ErrInvalidConfig)
	}
}

func TestClientConfigKeepsDisabledPolicies(t *testing.T) {
	got, err := normalizeClientConfig(ClientConfig{
		Retry:     RetryPolicy{Disabled: true},
		RateLimit: RateLimitPolicy{Disabled: true},
	})
	require.NoError(t, err)
	assert.True(t, got.Retry.Disabled)
	assert.True(t, got.RateLimit.Disabled)
}
