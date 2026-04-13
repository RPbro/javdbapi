package javdbapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	fixtureSearchPath  = "/search"
	fixtureVideoPath   = "/v/abc123"
	fixtureReviewsPath = fixtureVideoPath + "/reviews/lastest"
)

func TestListEndpoints(t *testing.T) {
	server, stats := newListOnlyFixtureServer(t)

	c, err := NewClient(Config{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})
	require.NoError(t, err)

	ctx := context.Background()

	cases := []struct {
		name string
		run  func(context.Context) ([]Video, error)
	}{
		{
			name: "home",
			run: func(ctx context.Context) ([]Video, error) {
				return c.Home(ctx, HomeQuery{
					Type:   HomeTypeAll,
					Filter: HomeFilterAll,
					Sort:   HomeSortPublishDate,
					Page:   1,
				})
			},
		},
		{
			name: "search",
			run: func(ctx context.Context) ([]Video, error) {
				return c.Search(ctx, SearchQuery{Keyword: "abc", Page: 1})
			},
		},
		{
			name: "maker",
			run: func(ctx context.Context) ([]Video, error) {
				return c.Maker(ctx, MakerQuery{MakerID: "7R", Filter: MakerFilterAll, Page: 1})
			},
		},
		{
			name: "actor",
			run: func(ctx context.Context) ([]Video, error) {
				return c.Actor(ctx, ActorQuery{ActorID: "neRNX", Filters: []ActorFilter{ActorFilterAll}, Page: 1})
			},
		},
		{
			name: "ranking",
			run: func(ctx context.Context) ([]Video, error) {
				return c.Ranking(ctx, RankingQuery{Period: RankingPeriodDaily, Type: RankingTypeCensored, Page: 1})
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			videos, err := tc.run(ctx)
			require.NoError(t, err)
			require.Len(t, videos, 1)

			video := videos[0]
			assert.Equal(t, fixtureVideoPath, video.ID)
			assert.Equal(t, "Sample Title", video.Title)
			assert.Equal(t, "ABC-123", video.Code)
			assert.Contains(t, video.URL, fixtureVideoPath)
			assert.Equal(t, "https://img.example/cover.jpg", video.CoverURL)
			assert.Equal(t, time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), video.PublishedAt)
			assert.Equal(t, 4.1, video.Score)
			assert.Equal(t, 22, video.ScoreCount)
			assert.True(t, video.HasSubtitle)
			assert.Empty(t, video.PreviewURL)
			assert.Empty(t, video.Actors)
			assert.Empty(t, video.Tags)
			assert.Empty(t, video.Screenshots)
			assert.Empty(t, video.Magnets)
			assert.Empty(t, video.Reviews)
			assert.Zero(t, stats.detailRequests)
			assert.Zero(t, stats.reviewRequests)
		})
	}

	t.Run("invalid query", func(t *testing.T) {
		invalidCases := []struct {
			name string
			run  func(context.Context) error
		}{
			{
				name: "search missing keyword",
				run: func(ctx context.Context) error {
					_, err := c.Search(ctx, SearchQuery{})
					return err
				},
			},
			{
				name: "maker missing id",
				run: func(ctx context.Context) error {
					_, err := c.Maker(ctx, MakerQuery{})
					return err
				},
			},
			{
				name: "actor missing id",
				run: func(ctx context.Context) error {
					_, err := c.Actor(ctx, ActorQuery{})
					return err
				},
			},
		}

		for _, tc := range invalidCases {
			t.Run(tc.name, func(t *testing.T) {
				err := tc.run(ctx)
				require.Error(t, err)
				assert.True(t, errors.Is(err, ErrInvalidQuery))
			})
		}
	})

	t.Run("empty list dom returns empty result", func(t *testing.T) {
		emptyListHTML := []byte(`<html><body><div class="empty-message">暫無內容</div></body></html>`)

		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case fixtureSearchPath:
				_, _ = w.Write(emptyListHTML)
			default:
				http.NotFound(w, r)
			}
		})
		srv := httptest.NewServer(mux)
		t.Cleanup(srv.Close)

		c2, err := NewClient(Config{BaseURL: srv.URL, Timeout: 5 * time.Second})
		require.NoError(t, err)

		_, err = c2.Search(ctx, SearchQuery{Keyword: "__codex_nohit_20260413__", Page: 1})
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrEmptyResult))
	})

	t.Run("non target list dom returns error", func(t *testing.T) {
		nonTargetHTML := []byte(`<html><body><div class="login-wall">please login</div></body></html>`)

		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case fixtureSearchPath:
				_, _ = w.Write(nonTargetHTML)
			default:
				http.NotFound(w, r)
			}
		})
		srv := httptest.NewServer(mux)
		t.Cleanup(srv.Close)

		c2, err := NewClient(Config{BaseURL: srv.URL, Timeout: 5 * time.Second})
		require.NoError(t, err)

		_, err = c2.Search(ctx, SearchQuery{Keyword: "__codex_nontarget_20260413__", Page: 1})
		require.Error(t, err)
		assert.False(t, errors.Is(err, ErrEmptyResult))
	})
}

type requestStats struct {
	detailRequests int
	reviewRequests int
}

func TestVideoEndpoint(t *testing.T) {
	server := newFixtureServer(t)

	c, err := NewClient(Config{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})
	require.NoError(t, err)

	ctx := context.Background()

	video, err := c.Video(ctx, VideoQuery{Path: fixtureVideoPath})
	require.NoError(t, err)
	require.NotNil(t, video)

	assert.Equal(t, fixtureVideoPath, video.ID)
	assert.Equal(t, "Sample Title", video.Title)
	assert.Equal(t, "ABC-123", video.Code)
	assert.NotEmpty(t, video.URL)
	assert.Equal(t, "https://img.example/detail.jpg", video.CoverURL)
	assert.Equal(t, time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), video.PublishedAt)
	assert.Equal(t, 4.1, video.Score)
	assert.Equal(t, 22, video.ScoreCount)
	assert.True(t, video.HasSubtitle)
	assert.Equal(t, "https://cdn.example/trailer.mp4", video.PreviewURL)
	assert.Equal(t, []string{"Julia"}, video.Actors)
	assert.Equal(t, []string{"Drama", "中文字幕"}, video.Tags)
	assert.Equal(t, []string{"https://img.example/shot-1.jpg"}, video.Screenshots)
	assert.Equal(t, []string{"magnet:?xt=urn:btih:1234567890abcdef1234567890abcdef12345678"}, video.Magnets)
	assert.Len(t, video.Reviews, 1)

	t.Run("invalid query", func(t *testing.T) {
		_, err := c.Video(ctx, VideoQuery{})
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidQuery))
	})

	t.Run("empty reviews still returns video", func(t *testing.T) {
		videoHTML, err := os.ReadFile(filepath.Join("internal", "testdata", "video.html"))
		require.NoError(t, err)

		emptyReviewsHTML := []byte(`<html><body><article class="message video-panel"><div class="message-body">暫無內容, 標記「看過」即可對影片進行短評</div></article></body></html>`)

		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case fixtureVideoPath:
				_, _ = w.Write(videoHTML)
			case fixtureReviewsPath:
				_, _ = w.Write(emptyReviewsHTML)
			default:
				http.NotFound(w, r)
			}
		})
		srv := httptest.NewServer(mux)
		t.Cleanup(srv.Close)

		c2, err := NewClient(Config{BaseURL: srv.URL, Timeout: 5 * time.Second})
		require.NoError(t, err)

		video, err := c2.Video(ctx, VideoQuery{Path: fixtureVideoPath})
		require.NoError(t, err)
		require.NotNil(t, video)
		assert.Empty(t, video.Reviews)
	})

	t.Run("non target reviews dom returns error", func(t *testing.T) {
		videoHTML, err := os.ReadFile(filepath.Join("internal", "testdata", "video.html"))
		require.NoError(t, err)

		nonTargetReviewsHTML := []byte(`<html><body><div class="login-wall">please login</div></body></html>`)

		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case fixtureVideoPath:
				_, _ = w.Write(videoHTML)
			case fixtureReviewsPath:
				_, _ = w.Write(nonTargetReviewsHTML)
			default:
				http.NotFound(w, r)
			}
		})
		srv := httptest.NewServer(mux)
		t.Cleanup(srv.Close)

		c2, err := NewClient(Config{BaseURL: srv.URL, Timeout: 5 * time.Second})
		require.NoError(t, err)

		_, err = c2.Video(ctx, VideoQuery{Path: fixtureVideoPath})
		require.Error(t, err)
	})

	t.Run("url mode uses same host for reviews", func(t *testing.T) {
		videoHTML, err := os.ReadFile(filepath.Join("internal", "testdata", "video.html"))
		require.NoError(t, err)

		reviewsHTML, err := os.ReadFile(filepath.Join("internal", "testdata", "reviews.html"))
		require.NoError(t, err)

		detailsCount := 0
		reviewsCount := 0
		muxA := http.NewServeMux()
		muxA.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case fixtureVideoPath:
				detailsCount++
				_, _ = w.Write(videoHTML)
			case fixtureReviewsPath:
				reviewsCount++
				_, _ = w.Write(reviewsHTML)
			default:
				http.NotFound(w, r)
			}
		})
		srvA := httptest.NewServer(muxA)
		t.Cleanup(srvA.Close)

		badReviewsCount := 0
		muxB := http.NewServeMux()
		muxB.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == fixtureReviewsPath {
				badReviewsCount++
				http.Error(w, "bad host", http.StatusInternalServerError)
				return
			}
			http.NotFound(w, r)
		})
		srvB := httptest.NewServer(muxB)
		t.Cleanup(srvB.Close)

		c3, err := NewClient(Config{BaseURL: srvB.URL, Timeout: 5 * time.Second})
		require.NoError(t, err)

		video, err := c3.Video(ctx, VideoQuery{URL: srvA.URL + fixtureVideoPath})
		require.NoError(t, err)
		require.NotNil(t, video)
		assert.Len(t, video.Reviews, 1)
		assert.Equal(t, 1, detailsCount)
		assert.Equal(t, 1, reviewsCount)
		assert.Equal(t, 0, badReviewsCount)
	})
}

func newFixtureServer(t *testing.T) *httptest.Server {
	t.Helper()

	listHTML, err := os.ReadFile(filepath.Join("internal", "testdata", "list.html"))
	require.NoError(t, err)

	videoHTML, err := os.ReadFile(filepath.Join("internal", "testdata", "video.html"))
	require.NoError(t, err)

	reviewsHTML, err := os.ReadFile(filepath.Join("internal", "testdata", "reviews.html"))
	require.NoError(t, err)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/", fixtureSearchPath, "/makers/7R", "/actors/neRNX", "/rankings/movies":
			_, _ = w.Write(listHTML)
		case fixtureVideoPath:
			_, _ = w.Write(videoHTML)
		case fixtureReviewsPath:
			_, _ = w.Write(reviewsHTML)
		default:
			http.NotFound(w, r)
		}
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func newListOnlyFixtureServer(t *testing.T) (*httptest.Server, *requestStats) {
	t.Helper()

	listHTML, err := os.ReadFile(filepath.Join("internal", "testdata", "list.html"))
	require.NoError(t, err)

	stats := &requestStats{}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/", fixtureSearchPath, "/makers/7R", "/actors/neRNX", "/rankings/movies":
			_, _ = w.Write(listHTML)
		case fixtureVideoPath:
			stats.detailRequests++
			http.Error(w, "detail should not be requested", http.StatusTooManyRequests)
		case fixtureReviewsPath:
			stats.reviewRequests++
			http.Error(w, "reviews should not be requested", http.StatusTooManyRequests)
		default:
			http.NotFound(w, r)
		}
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server, stats
}
