package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	javdbapi "github.com/RPbro/javdbapi"
)

func TestSearchCommandUsesCookieFile(t *testing.T) {
	list, err := os.ReadFile("../../internal/scrape/testdata/list.html")
	require.NoError(t, err)
	const session = "test-search-session"
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search" {
			t.Errorf("unexpected request to %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		cookie, err := r.Cookie("session")
		if err != nil || cookie.Value != session {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write(list)
	}))
	t.Cleanup(server.Close)

	// Use the test server's trusted transport for the real CLI client builder.
	previousTransport := http.DefaultTransport
	http.DefaultTransport = server.Client().Transport
	t.Cleanup(func() { http.DefaultTransport = previousTransport })
	args := []string{
		"javdbapi", "search", "--keyword", "fixture", "--base-url", server.URL,
		"--summary-only", "--output", "console",
	}

	cmd := newCommand(buildRealFetcher, io.Discard, io.Discard)
	err = cmd.Run(context.Background(), args)
	require.ErrorIs(t, err, javdbapi.ErrAuthenticationRequired)

	filename := filepath.Join(t.TempDir(), "cookie")
	require.NoError(t, os.WriteFile(filename, []byte("session="+session+"\n"), 0o600))
	var stdout bytes.Buffer
	cmd = newCommand(buildRealFetcher, &stdout, io.Discard)
	require.NoError(t, cmd.Run(context.Background(), append(args, "--cookie-file", filename)))
	assert.Contains(t, stdout.String(), `"code":`)
}

func TestSearchCommandCookieFileErrorOmitsCredentials(t *testing.T) {
	t.Parallel()

	const session = "test-search-session"
	filename := filepath.Join(t.TempDir(), "cookie")
	require.NoError(t, os.WriteFile(filename, []byte("Cookie: session="+session), 0o600))
	cmd := newCommand(buildRealFetcher, io.Discard, io.Discard)
	err := cmd.Run(context.Background(), []string{
		"javdbapi", "search", "--keyword", "fixture", "--cookie-file", filename,
	})
	require.ErrorContains(t, err, "invalid --cookie-file")
	assert.NotContains(t, err.Error(), session)
}
