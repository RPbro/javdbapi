package cliapp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	javdbapi "github.com/RPbro/javdbapi"
	"github.com/RPbro/javdbapi/internal/clioutput"
	"github.com/RPbro/javdbapi/internal/web"
)

type fakeFetcher struct {
	searchPages map[int][]javdbapi.Video
	listErrs    map[int]error
	videoByPath map[string]*javdbapi.Video
	videoErrs   map[string]error
	videoCalls  []string
	searchFn    func(javdbapi.SearchQuery) ([]javdbapi.Video, error)
	videoFn     func(javdbapi.VideoQuery) (*javdbapi.Video, error)
}

func (f *fakeFetcher) Home(context.Context, javdbapi.HomeQuery) ([]javdbapi.Video, error) {
	return nil, fmt.Errorf("unexpected Home call")
}

func (f *fakeFetcher) Search(_ context.Context, q javdbapi.SearchQuery) ([]javdbapi.Video, error) {
	if f.searchFn != nil {
		return f.searchFn(q)
	}
	if err := f.listErrs[q.Page]; err != nil {
		return nil, err
	}
	return f.searchPages[q.Page], nil
}

func (f *fakeFetcher) Maker(context.Context, javdbapi.MakerQuery) ([]javdbapi.Video, error) {
	return nil, fmt.Errorf("unexpected Maker call")
}

func (f *fakeFetcher) Actor(context.Context, javdbapi.ActorQuery) ([]javdbapi.Video, error) {
	return nil, fmt.Errorf("unexpected Actor call")
}

func (f *fakeFetcher) Ranking(context.Context, javdbapi.RankingQuery) ([]javdbapi.Video, error) {
	return nil, fmt.Errorf("unexpected Ranking call")
}

func (f *fakeFetcher) Video(_ context.Context, q javdbapi.VideoQuery) (*javdbapi.Video, error) {
	if f.videoFn != nil {
		return f.videoFn(q)
	}
	path := q.Path
	f.videoCalls = append(f.videoCalls, path)
	if err := f.videoErrs[path]; err != nil {
		return nil, err
	}
	return f.videoByPath[path], nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func testLoggerTo(w io.Writer) *slog.Logger {
	return slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func rateLimitedError(rawURL string) error {
	return fmt.Errorf("%w: %w", javdbapi.ErrUnexpectedStatus, &web.UnexpectedStatusError{
		StatusCode: 429,
		URL:        rawURL,
	})
}

func TestRunListCommandStopsOnEmptyResultSkipsFreshAndDeduplicates(t *testing.T) {
	now := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	dir := t.TempDir()
	store := clioutput.NewStore(dir, func() time.Time { return now })

	freshDoc := clioutput.Document{
		Metadata: clioutput.Metadata{
			LastUpdated: now.Add(-1 * time.Hour),
			Path:        "/v/fresh",
			PathKey:     "fresh",
			Sources:     []clioutput.Source{clioutput.NewSearchSource("VR", 1)},
		},
		Video: javdbapi.Video{ID: "/v/fresh", Code: "FRESH-001"},
	}
	require.NoError(t, store.WriteFile(freshDoc))

	fetcher := &fakeFetcher{
		searchPages: map[int][]javdbapi.Video{
			1: {
				{ID: "/v/fresh", Code: "FRESH-001", Title: "fresh"},
				{ID: "/v/stale", Code: "STALE-001", Title: "stale"},
			},
			2: {
				{ID: "/v/stale", Code: "STALE-001", Title: "stale"},
				{ID: "/v/new", Code: "NEW-001", Title: "new"},
			},
		},
		listErrs: map[int]error{
			3: fmt.Errorf("%w: page 3", javdbapi.ErrEmptyResult),
		},
		videoByPath: map[string]*javdbapi.Video{
			"/v/stale": {ID: "/v/stale", Code: "STALE-001", Title: "stale"},
			"/v/new":   {ID: "/v/new", Code: "NEW-001", Title: "new"},
		},
	}

	var stdout bytes.Buffer
	summary, err := RunListCommand(context.Background(), fetcher, store, ListRequest{
		Shared: SharedOptions{
			OutputMode: OutputFile,
			OutputDir:  dir,
			StaleAfter: 24 * time.Hour,
			Stdout:     &stdout,
			Logger:     testLogger(),
		},
		Command:  CommandSearch,
		Page:     1,
		MaxPages: 3,
		Search:   &javdbapi.SearchQuery{Keyword: "VR"},
	})
	require.NoError(t, err)
	assert.Equal(t, Summary{
		PagesScanned: 2,
		Candidates:   4,
		Deduplicated: 3,
		Fetched:      2,
		SkippedFresh: 1,
		Failed:       0,
	}, summary)
	assert.Equal(t, []string{"/v/stale", "/v/new"}, fetcher.videoCalls)
	_, err = os.Stat(filepath.Join(dir, "new.json"))
	require.NoError(t, err)
}

func TestRunListCommandReturnsListErrors(t *testing.T) {
	t.Parallel()

	store := clioutput.NewStore(t.TempDir(), time.Now)
	fetcher := &fakeFetcher{
		listErrs: map[int]error{
			1: fmt.Errorf("search page 1: %w", os.ErrPermission),
		},
	}

	_, err := RunListCommand(context.Background(), fetcher, store, ListRequest{
		Shared: SharedOptions{
			OutputMode: OutputFile,
			OutputDir:  t.TempDir(),
			StaleAfter: 24 * time.Hour,
			Stdout:     io.Discard,
			Logger:     testLogger(),
		},
		Command:  CommandSearch,
		Page:     1,
		MaxPages: 1,
		Search:   &javdbapi.SearchQuery{Keyword: "VR"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "search page 1")
}

func TestRunListCommandFailFastStopsAfterFirstVideoError(t *testing.T) {
	t.Parallel()

	store := clioutput.NewStore(t.TempDir(), time.Now)
	fetcher := &fakeFetcher{
		searchPages: map[int][]javdbapi.Video{
			1: {
				{ID: "/v/bad", Code: "BAD-001", Title: "bad"},
				{ID: "/v/never", Code: "NEVER-001", Title: "never"},
			},
		},
		videoErrs: map[string]error{
			"/v/bad": fmt.Errorf("detail fetch failed"),
		},
	}

	summary, err := RunListCommand(context.Background(), fetcher, store, ListRequest{
		Shared: SharedOptions{
			OutputMode: OutputFile,
			OutputDir:  t.TempDir(),
			StaleAfter: 24 * time.Hour,
			Stdout:     io.Discard,
			Logger:     testLogger(),
			FailFast:   true,
		},
		Command:  CommandSearch,
		Page:     1,
		MaxPages: 1,
		Search:   &javdbapi.SearchQuery{Keyword: "VR"},
	})
	require.Error(t, err)
	assert.Equal(t, Summary{PagesScanned: 1, Candidates: 2, Deduplicated: 2, Failed: 1}, summary)
	assert.Equal(t, []string{"/v/bad"}, fetcher.videoCalls)
}

func TestRunVideoCommandRejectsCanonicalPathMismatch(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := clioutput.NewStore(dir, time.Now)
	fetcher := &fakeFetcher{
		videoByPath: map[string]*javdbapi.Video{
			"/v/input": {ID: "/v/redirected", Code: "REDIR-001", Title: "redirected"},
		},
	}

	var stdout bytes.Buffer
	summary, err := RunVideoCommand(context.Background(), fetcher, store, VideoRequest{
		Shared: SharedOptions{
			OutputMode: OutputBoth,
			OutputDir:  dir,
			StaleAfter: 24 * time.Hour,
			Stdout:     &stdout,
			Logger:     testLogger(),
		},
		Path: "/v/input",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "canonical path mismatch")
	assert.Equal(t, Summary{Failed: 1}, summary)
	assert.Empty(t, stdout.String())
	_, statErr := os.Stat(filepath.Join(dir, "redirected.json"))
	assert.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestRunVideoCommandConsoleModeSkipsMissingCacheDir(t *testing.T) {
	t.Parallel()

	cacheDir := filepath.Join(t.TempDir(), "missing-cache")
	store := clioutput.NewStore(cacheDir, time.Now)
	fetcher := &fakeFetcher{
		videoByPath: map[string]*javdbapi.Video{
			"/v/ZNdEbV": {ID: "/v/ZNdEbV", Code: "ABC-123", Title: "sample"},
		},
	}

	var stdout bytes.Buffer
	summary, err := RunVideoCommand(context.Background(), fetcher, store, VideoRequest{
		Shared: SharedOptions{
			OutputMode: OutputConsole,
			OutputDir:  cacheDir,
			StaleAfter: 24 * time.Hour,
			BaseURL:    "https://javdb.com",
			Stdout:     &stdout,
			Logger:     testLogger(),
		},
		URL: "https://javdb.com/v/ZNdEbV?locale=zh",
	})
	require.NoError(t, err)
	assert.Equal(t, Summary{Fetched: 1}, summary)
	assert.Contains(t, stdout.String(), `"path":"/v/ZNdEbV"`)
	_, statErr := os.Stat(cacheDir)
	assert.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestRunListCommandRejectsInvalidParams(t *testing.T) {
	t.Parallel()

	store := clioutput.NewStore(t.TempDir(), time.Now)
	fetcher := &fakeFetcher{}

	cases := []struct {
		name    string
		req     ListRequest
		wantErr string
	}{
		{
			name: "max_pages_zero",
			req: ListRequest{
				Shared:   SharedOptions{OutputMode: OutputFile, OutputDir: t.TempDir(), Stdout: io.Discard, Logger: testLogger()},
				Command:  CommandSearch,
				Page:     1,
				MaxPages: 0,
				Search:   &javdbapi.SearchQuery{Keyword: "VR"},
			},
			wantErr: "invalid --max-pages 0",
		},
		{
			name: "page_zero",
			req: ListRequest{
				Shared:   SharedOptions{OutputMode: OutputFile, OutputDir: t.TempDir(), Stdout: io.Discard, Logger: testLogger()},
				Command:  CommandSearch,
				Page:     0,
				MaxPages: 1,
				Search:   &javdbapi.SearchQuery{Keyword: "VR"},
			},
			wantErr: "invalid --page 0",
		},
		{
			name: "invalid_output_mode",
			req: ListRequest{
				Shared:   SharedOptions{OutputMode: "invalid", OutputDir: t.TempDir(), Stdout: io.Discard, Logger: testLogger()},
				Command:  CommandSearch,
				Page:     1,
				MaxPages: 1,
				Search:   &javdbapi.SearchQuery{Keyword: "VR"},
			},
			wantErr: `invalid --output "invalid"`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := RunListCommand(context.Background(), fetcher, store, tc.req)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

func TestRunVideoCommandRejectsInvalidOutputMode(t *testing.T) {
	t.Parallel()

	store := clioutput.NewStore(t.TempDir(), time.Now)
	_, err := RunVideoCommand(context.Background(), &fakeFetcher{}, store, VideoRequest{
		Shared: SharedOptions{OutputMode: "bad", OutputDir: t.TempDir(), Stdout: io.Discard, Logger: testLogger()},
		Path:   "/v/ZNdEbV",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `invalid --output "bad"`)
}

func TestRunVideoCommandRejectsURLFromDifferentHost(t *testing.T) {
	t.Parallel()

	store := clioutput.NewStore(t.TempDir(), time.Now)
	fetcher := &fakeFetcher{}

	_, err := RunVideoCommand(context.Background(), fetcher, store, VideoRequest{
		Shared: SharedOptions{
			OutputMode: OutputConsole,
			OutputDir:  t.TempDir(),
			StaleAfter: 24 * time.Hour,
			BaseURL:    "https://javdb.com",
			Stdout:     io.Discard,
			Logger:     testLogger(),
		},
		URL: "https://example.com/v/ZNdEbV",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must match --base-url host")
	assert.Empty(t, fetcher.videoCalls)
}

func TestRunVideoCommandSkipsFetchLogWhenFresh(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	dir := t.TempDir()
	store := clioutput.NewStore(dir, func() time.Time { return now })
	require.NoError(t, store.WriteFile(clioutput.Document{
		Metadata: clioutput.Metadata{
			LastUpdated: now.Add(-1 * time.Hour),
			Path:        "/v/fresh",
			PathKey:     "fresh",
			Sources:     []clioutput.Source{clioutput.NewVideoSource("/v/fresh")},
		},
		Video: javdbapi.Video{ID: "/v/fresh", Code: "FRESH-001"},
	}))

	var logs bytes.Buffer
	fetcher := &fakeFetcher{}

	summary, err := RunVideoCommand(context.Background(), fetcher, store, VideoRequest{
		Shared: SharedOptions{
			OutputMode: OutputConsole,
			OutputDir:  dir,
			StaleAfter: 24 * time.Hour,
			BaseURL:    "https://javdb.com",
			Stdout:     io.Discard,
			Logger:     testLoggerTo(&logs),
		},
		Path: "/v/fresh",
	})
	require.NoError(t, err)
	assert.Equal(t, Summary{SkippedFresh: 1}, summary)
	assert.Empty(t, fetcher.videoCalls)
	assert.NotContains(t, logs.String(), "fetching video")
}

func TestRunVideoCommandRetriesRateLimitThenSucceeds(t *testing.T) {
	t.Parallel()

	store := clioutput.NewStore(t.TempDir(), time.Now)
	attempts := 0
	fetcher := &fakeFetcher{
		videoFn: func(q javdbapi.VideoQuery) (*javdbapi.Video, error) {
			attempts++
			if attempts < 3 {
				return nil, rateLimitedError("https://javdb.com" + q.Path)
			}
			return &javdbapi.Video{ID: q.Path, Code: "ABC-123", Title: "sample"}, nil
		},
	}

	summary, err := RunVideoCommand(context.Background(), fetcher, store, VideoRequest{
		Shared: SharedOptions{
			OutputMode: OutputConsole,
			OutputDir:  t.TempDir(),
			StaleAfter: 24 * time.Hour,
			BaseURL:    "https://javdb.com",
			Delay:      time.Nanosecond,
			Stdout:     io.Discard,
			Logger:     testLogger(),
		},
		Path: "/v/ZNdEbV",
	})
	require.NoError(t, err)
	assert.Equal(t, 3, attempts)
	assert.Equal(t, Summary{Fetched: 1}, summary)
}

func TestRunListCommandRetriesRateLimitedListPage(t *testing.T) {
	t.Parallel()

	store := clioutput.NewStore(t.TempDir(), time.Now)
	listAttempts := 0
	videoAttempts := 0
	fetcher := &fakeFetcher{
		searchFn: func(q javdbapi.SearchQuery) ([]javdbapi.Video, error) {
			listAttempts++
			if listAttempts < 3 {
				return nil, rateLimitedError("https://javdb.com/search?page=1")
			}
			return []javdbapi.Video{{ID: "/v/ZNdEbV", Code: "ABC-123", Title: "sample"}}, nil
		},
		videoFn: func(q javdbapi.VideoQuery) (*javdbapi.Video, error) {
			videoAttempts++
			return &javdbapi.Video{ID: q.Path, Code: "ABC-123", Title: "sample"}, nil
		},
	}

	summary, err := RunListCommand(context.Background(), fetcher, store, ListRequest{
		Shared: SharedOptions{
			OutputMode: OutputConsole,
			OutputDir:  t.TempDir(),
			StaleAfter: 24 * time.Hour,
			Delay:      time.Nanosecond,
			Stdout:     io.Discard,
			Logger:     testLogger(),
		},
		Command:  CommandSearch,
		Page:     1,
		MaxPages: 1,
		Search:   &javdbapi.SearchQuery{Keyword: "VR"},
	})
	require.NoError(t, err)
	assert.Equal(t, 3, listAttempts)
	assert.Equal(t, 1, videoAttempts)
	assert.Equal(t, Summary{
		PagesScanned: 1,
		Candidates:   1,
		Deduplicated: 1,
		Fetched:      1,
	}, summary)
}
