package main

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveOnlySelectionSuccess(t *testing.T) {
	t.Parallel()

	selected, err := resolveOnlySelection("search,video")
	require.NoError(t, err)
	assert.Equal(t, []endpointName{endpointSearch, endpointVideo}, selected)
}

func TestResolveOnlySelectionUnknownEndpoint(t *testing.T) {
	t.Parallel()

	selected, err := resolveOnlySelection("search,unknown")
	require.Error(t, err)
	assert.Nil(t, selected)
}

func TestLoadSampleInputFromEnvDefaults(t *testing.T) {
	t.Setenv("JAVDB_SAMPLE_KEYWORD", "")
	t.Setenv("JAVDB_SAMPLE_MAKER_ID", "")
	t.Setenv("JAVDB_SAMPLE_ACTOR_ID", "")
	t.Setenv("JAVDB_SAMPLE_VIDEO_PATH", "")

	samples := loadSampleInputFromEnv()
	assert.Equal(t, "VR", samples.keyword)
	assert.Equal(t, "7R", samples.makerID)
	assert.Equal(t, "neRNX", samples.actorID)
	assert.Equal(t, "/v/ZNdEbV", samples.videoPath)
}

func TestLoadClientConfigFromEnv(t *testing.T) {
	t.Setenv("JAVDB_BASE_URL", "https://example.com")
	t.Setenv("JAVDB_PROXY_URL", "http://127.0.0.1:7890")
	t.Setenv("JAVDB_USER_AGENT", "ua-test")
	t.Setenv("JAVDB_TIMEOUT", "45s")

	cfg, timeout, err := loadClientConfigFromEnv()
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", cfg.BaseURL)
	assert.Equal(t, "http://127.0.0.1:7890", cfg.ProxyURL)
	assert.Equal(t, "ua-test", cfg.UserAgent)
	assert.Equal(t, 45*time.Second, cfg.Timeout)
	assert.Equal(t, 45*time.Second, timeout)
}

func TestLoadClientConfigFromEnvInvalidTimeout(t *testing.T) {
	t.Setenv("JAVDB_TIMEOUT", "abc")

	_, _, err := loadClientConfigFromEnv()
	require.Error(t, err)
}

func TestLoadClientConfigFromEnvNonPositiveTimeout(t *testing.T) {
	t.Setenv("JAVDB_TIMEOUT", "0s")

	_, _, err := loadClientConfigFromEnv()
	require.Error(t, err)
}

func TestIsTransientNetworkError(t *testing.T) {
	t.Parallel()

	timeoutErr := &net.DNSError{IsTimeout: true}
	assert.True(t, isTransientNetworkError(timeoutErr))
	assert.True(t, isTransientNetworkError(fmt.Errorf("read failed: connection reset by peer")))
	assert.False(t, isTransientNetworkError(fmt.Errorf("dial tcp: lookup javdb.com: no such host")))
	assert.False(t, isTransientNetworkError(fmt.Errorf("dial tcp 1.2.3.4:443: connect: connection refused")))
}

func TestResolveOnlyInputFlagTakesPriorityEvenWhenEmpty(t *testing.T) {
	t.Parallel()

	selected := resolveOnlyInput("", true, "search,video")
	assert.Equal(t, "", selected)
}

func TestRunHelpReturnsSuccess(t *testing.T) {
	exitCode := run([]string{"-h"})
	assert.Equal(t, 0, exitCode)
}

func TestExecuteWithRetryBoundedByTotalTimeout(t *testing.T) {
	t.Parallel()

	timeout := 120 * time.Millisecond
	start := time.Now()
	attempts := 0

	err := executeWithRetry(timeout, func(ctx context.Context) error {
		attempts++
		if attempts == 1 {
			return fmt.Errorf("read timeout")
		}
		<-ctx.Done()
		return ctx.Err()
	})
	require.Error(t, err)

	elapsed := time.Since(start)
	assert.Less(t, elapsed, 2*timeout)
}
