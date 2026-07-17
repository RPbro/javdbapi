package cliapp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	javdbapi "github.com/RPbro/javdbapi"
	"github.com/RPbro/javdbapi/internal/clioutput"
)

// fakeFetcher is a hand-rolled Fetcher stand-in. Detail/Reviews calls are
// recorded under a mutex since the worker pool calls them concurrently.
type fakeFetcher struct {
	pages    map[int]javdbapi.Page[javdbapi.VideoSummary]
	pageErrs map[int]error

	detailByID map[javdbapi.VideoID]*javdbapi.VideoDetail
	detailErrs map[javdbapi.VideoID]error
	detailFn   func(javdbapi.VideoID) (*javdbapi.VideoDetail, error)

	reviewsByID map[javdbapi.VideoID][]javdbapi.Review
	reviewsErrs map[javdbapi.VideoID]error
	reviewsFn   func(javdbapi.VideoID) ([]javdbapi.Review, error)

	mu           sync.Mutex
	searchCalls  []int
	detailCalls  []javdbapi.VideoID
	reviewsCalls []javdbapi.VideoID
}

func (f *fakeFetcher) Home(context.Context, javdbapi.HomeQuery) (javdbapi.Page[javdbapi.VideoSummary], error) {
	return javdbapi.Page[javdbapi.VideoSummary]{}, fmt.Errorf("unexpected Home call")
}

func (f *fakeFetcher) Search(_ context.Context, q javdbapi.SearchQuery) (javdbapi.Page[javdbapi.VideoSummary], error) {
	f.mu.Lock()
	f.searchCalls = append(f.searchCalls, q.Page)
	f.mu.Unlock()
	if err := f.pageErrs[q.Page]; err != nil {
		return javdbapi.Page[javdbapi.VideoSummary]{}, err
	}
	return f.pages[q.Page], nil
}

func (f *fakeFetcher) MakerVideos(context.Context, javdbapi.MakerVideosQuery) (javdbapi.Page[javdbapi.VideoSummary], error) {
	return javdbapi.Page[javdbapi.VideoSummary]{}, fmt.Errorf("unexpected MakerVideos call")
}

func (f *fakeFetcher) ActorVideos(context.Context, javdbapi.ActorVideosQuery) (javdbapi.Page[javdbapi.VideoSummary], error) {
	return javdbapi.Page[javdbapi.VideoSummary]{}, fmt.Errorf("unexpected ActorVideos call")
}

func (f *fakeFetcher) Ranking(context.Context, javdbapi.RankingQuery) (javdbapi.Page[javdbapi.VideoSummary], error) {
	return javdbapi.Page[javdbapi.VideoSummary]{}, fmt.Errorf("unexpected Ranking call")
}

func (f *fakeFetcher) Detail(_ context.Context, id javdbapi.VideoID) (*javdbapi.VideoDetail, error) {
	f.mu.Lock()
	f.detailCalls = append(f.detailCalls, id)
	f.mu.Unlock()

	if f.detailFn != nil {
		return f.detailFn(id)
	}
	if err := f.detailErrs[id]; err != nil {
		return nil, err
	}
	if d, ok := f.detailByID[id]; ok {
		return d, nil
	}
	detail := javdbapi.NewVideoDetail(javdbapi.VideoSummary{ID: id})
	return &detail, nil
}

func (f *fakeFetcher) Reviews(_ context.Context, id javdbapi.VideoID) ([]javdbapi.Review, error) {
	f.mu.Lock()
	f.reviewsCalls = append(f.reviewsCalls, id)
	f.mu.Unlock()

	if f.reviewsFn != nil {
		return f.reviewsFn(id)
	}
	if err := f.reviewsErrs[id]; err != nil {
		return nil, err
	}
	return f.reviewsByID[id], nil
}

func (f *fakeFetcher) detailCallCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.detailCalls)
}

func testLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestRunListCommandStopsAfterHasNextFalse(t *testing.T) {
	t.Parallel()

	store := clioutput.NewStore(t.TempDir(), time.Now)
	fetcher := &fakeFetcher{
		pages: map[int]javdbapi.Page[javdbapi.VideoSummary]{
			1: {Items: []javdbapi.VideoSummary{{ID: "a"}}, Number: 1, HasNext: false},
		},
	}

	summary, err := RunListCommand(context.Background(), fetcher, store, ListRequest{
		Shared: SharedOptions{
			OutputMode:  OutputConsole,
			OutputDir:   t.TempDir(),
			StaleAfter:  24 * time.Hour,
			Concurrency: 2,
			Stdout:      io.Discard,
			Logger:      testLogger(),
		},
		Command:  CommandSearch,
		Page:     1,
		MaxPages: 5,
		Search:   &javdbapi.SearchQuery{Keyword: "VR"},
	})
	require.NoError(t, err)
	assert.Equal(t, []int{1}, fetcher.searchCalls, "must not request a page after HasNext=false")
	assert.Equal(t, 1, summary.PagesScanned)
	assert.Equal(t, 1, summary.Fetched)
}

func TestRunListCommandDeduplicatesIDsAcrossPages(t *testing.T) {
	t.Parallel()

	store := clioutput.NewStore(t.TempDir(), time.Now)
	fetcher := &fakeFetcher{
		pages: map[int]javdbapi.Page[javdbapi.VideoSummary]{
			1: {Items: []javdbapi.VideoSummary{{ID: "a"}, {ID: "b"}}, Number: 1, HasNext: true},
			2: {Items: []javdbapi.VideoSummary{{ID: "b"}, {ID: "c"}}, Number: 2, HasNext: false},
		},
	}

	summary, err := RunListCommand(context.Background(), fetcher, store, ListRequest{
		Shared: SharedOptions{
			OutputMode:  OutputConsole,
			OutputDir:   t.TempDir(),
			StaleAfter:  24 * time.Hour,
			Concurrency: 2,
			Stdout:      io.Discard,
			Logger:      testLogger(),
		},
		Command:  CommandSearch,
		Page:     1,
		MaxPages: 5,
		Search:   &javdbapi.SearchQuery{Keyword: "VR"},
	})
	require.NoError(t, err)
	assert.Equal(t, Summary{PagesScanned: 2, Candidates: 4, Deduplicated: 3, Fetched: 3}, summary)
	assert.ElementsMatch(t, []javdbapi.VideoID{"a", "b", "c"}, fetcher.detailCalls)
}

func TestRunListCommandSummaryOnlySkipsDetailAndReviews(t *testing.T) {
	t.Parallel()

	store := clioutput.NewStore(t.TempDir(), time.Now)
	fetcher := &fakeFetcher{
		pages: map[int]javdbapi.Page[javdbapi.VideoSummary]{
			1: {Items: []javdbapi.VideoSummary{{ID: "a", Code: "AAA-001"}, {ID: "b", Code: "BBB-002"}}, Number: 1, HasNext: true},
			2: {Items: []javdbapi.VideoSummary{{ID: "b", Code: "BBB-002"}, {ID: "c", Code: "CCC-003"}}, Number: 2, HasNext: false},
		},
	}

	var stdout bytes.Buffer
	summary, err := RunListCommand(context.Background(), fetcher, store, ListRequest{
		Shared: SharedOptions{
			OutputMode:  OutputConsole,
			StaleAfter:  24 * time.Hour,
			Concurrency: 2,
			Stdout:      &stdout,
			Logger:      testLogger(),
		},
		Command:     CommandSearch,
		Page:        1,
		MaxPages:    5,
		SummaryOnly: true,
		Search:      &javdbapi.SearchQuery{Keyword: "VR"},
	})
	require.NoError(t, err)
	assert.Equal(t, Summary{PagesScanned: 2, Candidates: 4, Deduplicated: 3, SummariesOutput: 3}, summary)
	assert.Empty(t, fetcher.detailCalls, "summary-only must never call Detail")
	assert.Empty(t, fetcher.reviewsCalls, "summary-only must never call Reviews")

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	require.Len(t, lines, 3)
	assert.Contains(t, lines[0], `"id":"a"`)
	assert.Contains(t, lines[1], `"id":"b"`)
	assert.Contains(t, lines[2], `"id":"c"`)
	assert.NotContains(t, lines[0], `"score"`, "nil score must be omitted from summary-only output")
}

func TestRunListCommandSummaryOnlyRejectsNonConsoleOutput(t *testing.T) {
	t.Parallel()

	store := clioutput.NewStore(t.TempDir(), time.Now)
	fetcher := &fakeFetcher{}

	for _, mode := range []OutputMode{OutputFile, OutputBoth} {
		t.Run(string(mode), func(t *testing.T) {
			_, err := RunListCommand(context.Background(), fetcher, store, ListRequest{
				Shared: SharedOptions{
					OutputMode:  mode,
					OutputDir:   t.TempDir(),
					StaleAfter:  24 * time.Hour,
					Concurrency: 1,
					Stdout:      io.Discard,
					Logger:      testLogger(),
				},
				Command:     CommandSearch,
				Page:        1,
				MaxPages:    1,
				SummaryOnly: true,
				Search:      &javdbapi.SearchQuery{Keyword: "VR"},
			})
			require.Error(t, err)
			assert.Equal(t, "--summary-only requires --output console", err.Error())
		})
	}
}

func TestRunListCommandSummaryOnlyHasNoFilesystemSideEffects(t *testing.T) {
	t.Parallel()

	outputDir := filepath.Join(t.TempDir(), "does-not-exist-yet")
	store := clioutput.NewStore(outputDir, time.Now)
	fetcher := &fakeFetcher{
		pages: map[int]javdbapi.Page[javdbapi.VideoSummary]{
			1: {Items: []javdbapi.VideoSummary{{ID: "a"}}, Number: 1, HasNext: false},
		},
	}

	_, err := RunListCommand(context.Background(), fetcher, store, ListRequest{
		Shared: SharedOptions{
			OutputMode:  OutputConsole,
			OutputDir:   outputDir,
			StaleAfter:  24 * time.Hour,
			Concurrency: 1,
			Stdout:      io.Discard,
			Logger:      testLogger(),
		},
		Command:     CommandSearch,
		Page:        1,
		MaxPages:    1,
		SummaryOnly: true,
		Search:      &javdbapi.SearchQuery{Keyword: "VR"},
	})
	require.NoError(t, err)

	_, statErr := os.Stat(outputDir)
	assert.True(t, os.IsNotExist(statErr), "summary-only must not create the output dir")
}

func TestRunListCommandReturnsNonEmptyResultListErrors(t *testing.T) {
	t.Parallel()

	store := clioutput.NewStore(t.TempDir(), time.Now)
	wantErr := fmt.Errorf("boom")
	fetcher := &fakeFetcher{
		pageErrs: map[int]error{1: wantErr},
	}

	_, err := RunListCommand(context.Background(), fetcher, store, ListRequest{
		Shared: SharedOptions{
			OutputMode:  OutputConsole,
			OutputDir:   t.TempDir(),
			StaleAfter:  24 * time.Hour,
			Concurrency: 2,
			Stdout:      io.Discard,
			Logger:      testLogger(),
		},
		Command:  CommandSearch,
		Page:     1,
		MaxPages: 5,
		Search:   &javdbapi.SearchQuery{Keyword: "VR"},
	})
	require.ErrorIs(t, err, wantErr)
}

func TestRunListCommandConcurrencyIsBoundedAndPreservesDiscoveryOrder(t *testing.T) {
	t.Parallel()

	store := clioutput.NewStore(t.TempDir(), time.Now)

	ids := []javdbapi.VideoID{"a", "b", "c"}
	delays := map[javdbapi.VideoID]time.Duration{"a": 30 * time.Millisecond, "b": 20 * time.Millisecond, "c": 10 * time.Millisecond}

	var startWG sync.WaitGroup
	startWG.Add(len(ids))

	var active int32
	var maxActive int32
	var mu sync.Mutex

	fetcher := &fakeFetcher{
		pages: map[int]javdbapi.Page[javdbapi.VideoSummary]{
			1: {Items: []javdbapi.VideoSummary{{ID: "a"}, {ID: "b"}, {ID: "c"}}, Number: 1, HasNext: false},
		},
		detailFn: func(id javdbapi.VideoID) (*javdbapi.VideoDetail, error) {
			n := atomic.AddInt32(&active, 1)
			mu.Lock()
			if n > maxActive {
				maxActive = n
			}
			mu.Unlock()

			startWG.Done()
			startWG.Wait() // barrier: block until all 3 workers are in flight

			time.Sleep(delays[id])
			atomic.AddInt32(&active, -1)

			detail := javdbapi.NewVideoDetail(javdbapi.VideoSummary{ID: id})
			return &detail, nil
		},
	}

	var stdout bytes.Buffer
	summary, err := RunListCommand(context.Background(), fetcher, store, ListRequest{
		Shared: SharedOptions{
			OutputMode:  OutputConsole,
			OutputDir:   t.TempDir(),
			StaleAfter:  24 * time.Hour,
			Concurrency: 3,
			Stdout:      &stdout,
			Logger:      testLogger(),
		},
		Command:  CommandSearch,
		Page:     1,
		MaxPages: 1,
		Search:   &javdbapi.SearchQuery{Keyword: "VR"},
	})
	require.NoError(t, err)
	assert.Equal(t, int32(3), maxActive, "expected exactly 3 concurrent Detail calls")
	assert.Equal(t, 3, summary.Fetched)

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	require.Len(t, lines, 3)
	assert.Contains(t, lines[0], `"id":"a"`)
	assert.Contains(t, lines[1], `"id":"b"`)
	assert.Contains(t, lines[2], `"id":"c"`)
}

func TestRunListCommandFailFastCancelsRemainingWork(t *testing.T) {
	t.Parallel()

	store := clioutput.NewStore(t.TempDir(), time.Now)
	fetcher := &fakeFetcher{
		pages: map[int]javdbapi.Page[javdbapi.VideoSummary]{
			1: {Items: []javdbapi.VideoSummary{{ID: "a"}, {ID: "b"}}, Number: 1, HasNext: false},
		},
		detailErrs: map[javdbapi.VideoID]error{"a": fmt.Errorf("detail unavailable")},
	}

	_, err := RunListCommand(context.Background(), fetcher, store, ListRequest{
		Shared: SharedOptions{
			OutputMode:  OutputConsole,
			OutputDir:   t.TempDir(),
			StaleAfter:  24 * time.Hour,
			Concurrency: 1,
			FailFast:    true,
			Stdout:      io.Discard,
			Logger:      testLogger(),
		},
		Command:  CommandSearch,
		Page:     1,
		MaxPages: 1,
		Search:   &javdbapi.SearchQuery{Keyword: "VR"},
	})
	require.Error(t, err)
}

func TestRunListCommandRejectsInvalidParams(t *testing.T) {
	t.Parallel()

	store := clioutput.NewStore(t.TempDir(), time.Now)
	fetcher := &fakeFetcher{}
	base := ListRequest{
		Command:  CommandSearch,
		Page:     1,
		MaxPages: 1,
		Search:   &javdbapi.SearchQuery{Keyword: "VR"},
	}

	cases := []struct {
		name  string
		shape func(ListRequest) ListRequest
	}{
		{"invalid output mode", func(r ListRequest) ListRequest {
			r.Shared = SharedOptions{OutputMode: "bogus", Concurrency: 1, Stdout: io.Discard, Logger: testLogger()}
			return r
		}},
		{"invalid page", func(r ListRequest) ListRequest {
			r.Shared = SharedOptions{OutputMode: OutputConsole, Concurrency: 1, Stdout: io.Discard, Logger: testLogger()}
			r.Page = 0
			return r
		}},
		{"invalid max pages", func(r ListRequest) ListRequest {
			r.Shared = SharedOptions{OutputMode: OutputConsole, Concurrency: 1, Stdout: io.Discard, Logger: testLogger()}
			r.MaxPages = 0
			return r
		}},
		{"invalid concurrency", func(r ListRequest) ListRequest {
			r.Shared = SharedOptions{OutputMode: OutputConsole, Concurrency: 0, Stdout: io.Discard, Logger: testLogger()}
			return r
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := RunListCommand(context.Background(), fetcher, store, tc.shape(base))
			require.Error(t, err)
		})
	}
}

func TestRunVideoCommandRequiresID(t *testing.T) {
	t.Parallel()

	store := clioutput.NewStore(t.TempDir(), time.Now)
	fetcher := &fakeFetcher{}

	_, err := RunVideoCommand(context.Background(), fetcher, store, VideoRequest{
		Shared: SharedOptions{OutputMode: OutputConsole, Stdout: io.Discard, Logger: testLogger()},
	})
	require.Error(t, err)
}

func TestRunVideoCommandDoesNotProduceParseErrorForRealEmptyReviews(t *testing.T) {
	t.Parallel()

	detailFixture, err := os.ReadFile(filepath.Join("..", "scrape", "testdata", "detail-complete.html"))
	require.NoError(t, err)
	reviewsFixture, err := os.ReadFile(filepath.Join("..", "scrape", "testdata", "reviews-empty-message-body.html"))
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/reviews/lastest") {
			_, _ = w.Write(reviewsFixture)
			return
		}
		_, _ = w.Write(detailFixture)
	}))
	t.Cleanup(server.Close)

	client, err := javdbapi.NewClient(javdbapi.ClientConfig{
		BaseURL:   server.URL,
		Retry:     javdbapi.RetryPolicy{Disabled: true},
		RateLimit: javdbapi.RateLimitPolicy{Disabled: true},
	})
	require.NoError(t, err)

	dir := t.TempDir()
	store := clioutput.NewStore(dir, time.Now)

	summary, err := RunVideoCommand(context.Background(), ClientFetcher{Client: client}, store, VideoRequest{
		Shared: SharedOptions{OutputMode: OutputFile, OutputDir: dir, StaleAfter: 24 * time.Hour, Stdout: io.Discard, Logger: testLogger()},
		ID:     "P9Jkq9",
	})
	require.NoError(t, err)
	assert.Equal(t, Summary{Fetched: 1}, summary)

	state, err := store.Load("P9Jkq9", 24*time.Hour)
	require.NoError(t, err)
	require.NotNil(t, state.Document)
	assert.Empty(t, state.Document.PartialErrors, "the real empty-reviews notice must not be recorded as a partial parse_error")
	assert.Empty(t, state.Document.Reviews)
}

func TestRunVideoCommandSkipsFreshCache(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	now := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	store := clioutput.NewStore(dir, func() time.Time { return now })

	fetcher := &fakeFetcher{}
	firstOpts := VideoRequest{
		Shared: SharedOptions{OutputMode: OutputFile, OutputDir: dir, StaleAfter: 24 * time.Hour, Stdout: io.Discard, Logger: testLogger()},
		ID:     "P9Jkq9",
	}
	_, err := RunVideoCommand(context.Background(), fetcher, store, firstOpts)
	require.NoError(t, err)
	require.Equal(t, 1, fetcher.detailCallCount())

	summary, err := RunVideoCommand(context.Background(), fetcher, store, firstOpts)
	require.NoError(t, err)
	assert.Equal(t, Summary{SkippedFresh: 1}, summary)
	assert.Equal(t, 1, fetcher.detailCallCount(), "fresh cache must not re-fetch detail")
}

func TestRunVideoCommandConsoleModeSkipsMissingCacheDir(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "does-not-exist-yet")
	store := clioutput.NewStore(dir, time.Now)
	fetcher := &fakeFetcher{}

	var stdout bytes.Buffer
	summary, err := RunVideoCommand(context.Background(), fetcher, store, VideoRequest{
		Shared: SharedOptions{OutputMode: OutputConsole, OutputDir: dir, StaleAfter: 24 * time.Hour, Stdout: &stdout, Logger: testLogger()},
		ID:     "P9Jkq9",
	})
	require.NoError(t, err)
	assert.Equal(t, Summary{Fetched: 1}, summary)
	assert.Contains(t, stdout.String(), `"id":"P9Jkq9"`)

	_, statErr := os.Stat(dir)
	assert.True(t, os.IsNotExist(statErr), "console mode must not create the output dir just to check freshness")
}

func TestRunVideoCommandPersistsDetailDespiteReviewsFailureDefaultMode(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := clioutput.NewStore(dir, time.Now)
	fetcher := &fakeFetcher{
		reviewsErrs: map[javdbapi.VideoID]error{"P9Jkq9": fmt.Errorf("reviews unavailable")},
	}

	var stdout bytes.Buffer
	summary, err := RunVideoCommand(context.Background(), fetcher, store, VideoRequest{
		Shared: SharedOptions{OutputMode: OutputBoth, OutputDir: dir, StaleAfter: 24 * time.Hour, Stdout: &stdout, Logger: testLogger()},
		ID:     "P9Jkq9",
	})
	require.NoError(t, err)
	assert.Equal(t, Summary{PartialFailed: 1}, summary)

	state, err := store.Load("P9Jkq9", 24*time.Hour)
	require.NoError(t, err)
	require.NotNil(t, state.Document)
	require.Len(t, state.Document.PartialErrors, 1)
	assert.Equal(t, "reviews", state.Document.PartialErrors[0].Component)
	assert.Equal(t, "fetch_error", state.Document.PartialErrors[0].Kind)
	assert.Contains(t, stdout.String(), `"id":"P9Jkq9"`)
}

func TestRunVideoCommandMapsChallengeToStablePartialErrorKind(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := clioutput.NewStore(dir, time.Now)
	fetcher := &fakeFetcher{
		reviewsErrs: map[javdbapi.VideoID]error{"P9Jkq9": &javdbapi.OpError{Op: "reviews.fetch", Err: javdbapi.ErrChallenge}},
	}

	summary, err := RunVideoCommand(context.Background(), fetcher, store, VideoRequest{
		Shared: SharedOptions{OutputMode: OutputFile, OutputDir: dir, StaleAfter: 24 * time.Hour, Stdout: io.Discard, Logger: testLogger()},
		ID:     "P9Jkq9",
	})
	require.NoError(t, err)
	assert.Equal(t, Summary{PartialFailed: 1}, summary)

	state, err := store.Load("P9Jkq9", 24*time.Hour)
	require.NoError(t, err)
	require.NotNil(t, state.Document)
	require.Len(t, state.Document.PartialErrors, 1)
	assert.Equal(t, "challenge", state.Document.PartialErrors[0].Kind)
	assert.Equal(t, "reviews request was challenged", state.Document.PartialErrors[0].Message)
}

func TestRunVideoCommandMapsAuthenticationRequiredToStablePartialErrorKind(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := clioutput.NewStore(dir, time.Now)
	fetcher := &fakeFetcher{
		reviewsErrs: map[javdbapi.VideoID]error{
			"P9Jkq9": &javdbapi.OpError{Op: "reviews.fetch", Err: javdbapi.ErrAuthenticationRequired},
		},
	}

	summary, err := RunVideoCommand(context.Background(), fetcher, store, VideoRequest{
		Shared: SharedOptions{OutputMode: OutputFile, OutputDir: dir, StaleAfter: 24 * time.Hour, Stdout: io.Discard, Logger: testLogger()},
		ID:     "P9Jkq9",
	})
	require.NoError(t, err)
	assert.Equal(t, Summary{PartialFailed: 1}, summary)

	state, err := store.Load("P9Jkq9", 24*time.Hour)
	require.NoError(t, err)
	require.NotNil(t, state.Document)
	require.Len(t, state.Document.PartialErrors, 1)
	assert.Equal(t, "authentication_required", state.Document.PartialErrors[0].Kind)
	assert.Equal(t, "reviews request requires authentication", state.Document.PartialErrors[0].Message)
}

func TestRunVideoCommandPartialErrorMessageDoesNotLeakURL(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := clioutput.NewStore(dir, time.Now)
	leakyErr := &javdbapi.OpError{
		Op:  "reviews.fetch",
		URL: "https://javdb.com/v/P9Jkq9?q=secret-search-term",
		Err: javdbapi.ErrEmptyResult,
	}
	fetcher := &fakeFetcher{
		reviewsErrs: map[javdbapi.VideoID]error{"P9Jkq9": leakyErr},
	}

	summary, err := RunVideoCommand(context.Background(), fetcher, store, VideoRequest{
		Shared: SharedOptions{OutputMode: OutputFile, OutputDir: dir, StaleAfter: 24 * time.Hour, Stdout: io.Discard, Logger: testLogger()},
		ID:     "P9Jkq9",
	})
	require.NoError(t, err)
	assert.Equal(t, Summary{PartialFailed: 1}, summary)

	raw, err := os.ReadFile(filepath.Join(dir, "P9Jkq9.json"))
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "javdb.com")
	assert.NotContains(t, string(raw), "secret-search-term")

	state, err := store.Load("P9Jkq9", 24*time.Hour)
	require.NoError(t, err)
	require.NotNil(t, state.Document)
	require.Len(t, state.Document.PartialErrors, 1)
	assert.Equal(t, "empty_result", state.Document.PartialErrors[0].Kind)
	assert.NotContains(t, state.Document.PartialErrors[0].Message, "javdb.com")
}

func TestRunVideoCommandFailFastCancelsAndReturnsReviewsError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := clioutput.NewStore(dir, time.Now)
	wantErr := fmt.Errorf("reviews unavailable")
	fetcher := &fakeFetcher{
		reviewsErrs: map[javdbapi.VideoID]error{"P9Jkq9": wantErr},
	}

	summary, err := RunVideoCommand(context.Background(), fetcher, store, VideoRequest{
		Shared: SharedOptions{OutputMode: OutputFile, OutputDir: dir, StaleAfter: 24 * time.Hour, Stdout: io.Discard, Logger: testLogger(), FailFast: true},
		ID:     "P9Jkq9",
	})
	require.ErrorIs(t, err, wantErr)
	assert.Equal(t, Summary{PartialFailed: 1}, summary)

	_, statErr := os.Stat(filepath.Join(dir, "P9Jkq9.json"))
	require.NoError(t, statErr, "detail should still be persisted despite fail-fast reviews error")
}

func TestRunVideoCommandRetriesReviewsOnNextInvocationReusingFreshDetail(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	dir := t.TempDir()
	store := clioutput.NewStore(dir, func() time.Time { return now })

	fetcher := &fakeFetcher{
		reviewsErrs: map[javdbapi.VideoID]error{"P9Jkq9": fmt.Errorf("reviews unavailable")},
	}

	opts := SharedOptions{OutputMode: OutputFile, OutputDir: dir, StaleAfter: 24 * time.Hour, Stdout: io.Discard, Logger: testLogger()}

	_, err := RunVideoCommand(context.Background(), fetcher, store, VideoRequest{Shared: opts, ID: "P9Jkq9"})
	require.NoError(t, err)
	assert.Equal(t, 1, fetcher.detailCallCount())

	fetcher.reviewsErrs = nil
	fetcher.reviewsByID = map[javdbapi.VideoID][]javdbapi.Review{"P9Jkq9": {{ID: "r1"}}}

	summary, err := RunVideoCommand(context.Background(), fetcher, store, VideoRequest{Shared: opts, ID: "P9Jkq9"})
	require.NoError(t, err)
	assert.Equal(t, Summary{Fetched: 1}, summary)
	assert.Equal(t, 1, fetcher.detailCallCount(), "detail must not be re-fetched since it is still fresh")

	state, err := store.Load("P9Jkq9", 24*time.Hour)
	require.NoError(t, err)
	require.NotNil(t, state.Document)
	require.Len(t, state.Document.Reviews, 1)
	assert.Empty(t, state.Document.PartialErrors)
}
