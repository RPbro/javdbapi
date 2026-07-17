package javdbapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	javdbapi "github.com/RPbro/javdbapi"
)

func TestVideoSummaryScoreJSONOmitsWhenNil(t *testing.T) {
	raw, err := json.Marshal(javdbapi.VideoSummary{ID: "P9Jkq9", Code: "SNOS-177"})
	require.NoError(t, err)
	assert.NotContains(t, string(raw), `"score"`)
}

func TestVideoSummaryScoreJSONRoundTripsWhenPresent(t *testing.T) {
	summary := javdbapi.VideoSummary{ID: "P9Jkq9", Score: &javdbapi.Score{Value: 4.55, Count: 3080}}
	raw, err := json.Marshal(summary)
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"score":{"value":4.55,"count":3080}`)

	var decoded javdbapi.VideoSummary
	require.NoError(t, json.Unmarshal(raw, &decoded))
	require.NotNil(t, decoded.Score)
	assert.Equal(t, 4.55, decoded.Score.Value)
	assert.Equal(t, 3080, decoded.Score.Count)
}

func TestVideoSummaryDecodesLegacyScoreObjectIntoPointer(t *testing.T) {
	legacyJSON := `{"id":"P9Jkq9","code":"SNOS-177","title":"","published_at":"0001-01-01T00:00:00Z","score":{"value":3.2,"count":12},"availability":{"has_magnet":false,"has_subtitle_magnet":false,"is_playable":false}}`

	var decoded javdbapi.VideoSummary
	require.NoError(t, json.Unmarshal([]byte(legacyJSON), &decoded))
	require.NotNil(t, decoded.Score)
	assert.Equal(t, 3.2, decoded.Score.Value)
	assert.Equal(t, 12, decoded.Score.Count)
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("internal", "scrape", "testdata", name))
	require.NoError(t, err)
	return data
}

// newFixtureServer serves the given fixture body for every request and
// records each requested path (without query) in request order.
func newFixtureServer(t *testing.T, body []byte, onRequest func(path string)) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		onRequest(r.URL.Path)
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)
	return server
}

func newDetailFixtureServer(t *testing.T, onRequest func(path string)) *httptest.Server {
	t.Helper()
	return newFixtureServer(t, readFixture(t, "detail-complete.html"), onRequest)
}

func newReviewsFixtureServer(t *testing.T) *httptest.Server {
	t.Helper()
	return newFixtureServer(t, readFixture(t, "reviews.html"), func(string) {})
}

func newClientForServer(t *testing.T, server *httptest.Server, opts ...func(*javdbapi.ClientConfig)) *javdbapi.Client {
	t.Helper()
	cfg := javdbapi.ClientConfig{
		BaseURL:   server.URL,
		Retry:     javdbapi.RetryPolicy{Disabled: true},
		RateLimit: javdbapi.RateLimitPolicy{Disabled: true},
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	client, err := javdbapi.NewClient(cfg)
	require.NoError(t, err)
	return client
}

func TestClientDetailDoesNotFetchReviews(t *testing.T) {
	var paths []string
	server := newDetailFixtureServer(t, func(path string) { paths = append(paths, path) })
	client := newClientForServer(t, server)
	detail, err := client.Detail(context.Background(), "P9Jkq9")
	require.NoError(t, err)
	assert.Equal(t, javdbapi.VideoID("P9Jkq9"), detail.Summary.ID)
	assert.Equal(t, []string{"/v/P9Jkq9"}, paths)
}

func TestClientReviewsAreIndependent(t *testing.T) {
	client := newClientForServer(t, newReviewsFixtureServer(t))
	reviews, err := client.Reviews(context.Background(), "P9Jkq9")
	require.NoError(t, err)
	require.NotEmpty(t, reviews)
}

func TestClientReviewsRecognizesRealEmptyMessageBody(t *testing.T) {
	client := newClientForServer(t, newFixtureServer(t, readFixture(t, "reviews-empty-message-body.html"), func(string) {}))
	reviews, err := client.Reviews(context.Background(), "P9Jkq9")
	require.NoError(t, err, "the real no-reviews notice must not be classified as a parse failure")
	assert.NotNil(t, reviews)
	assert.Empty(t, reviews)
}

func TestClientMapsChallengeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("cf-mitigated", "challenge")
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)

	client := newClientForServer(t, server)
	_, err := client.Detail(context.Background(), "P9Jkq9")
	require.ErrorIs(t, err, javdbapi.ErrChallenge)
}

func TestClientMapsLoginRedirectToAuthenticationRequired(t *testing.T) {
	tests := map[string]func(context.Context, *javdbapi.Client) error{
		"detail": func(ctx context.Context, client *javdbapi.Client) error {
			_, err := client.Detail(ctx, "P9Jkq9")
			return err
		},
		"reviews": func(ctx context.Context, client *javdbapi.Client) error {
			_, err := client.Reviews(ctx, "P9Jkq9")
			return err
		},
	}

	for name, call := range tests {
		t.Run(name, func(t *testing.T) {
			var requestCount int
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestCount++
				http.Redirect(w, r, "/login?return_to=top-secret", http.StatusFound)
			}))
			t.Cleanup(server.Close)

			client := newClientForServer(t, server)
			err := call(context.Background(), client)
			require.ErrorIs(t, err, javdbapi.ErrAuthenticationRequired)
			assert.NotErrorIs(t, err, javdbapi.ErrParse)
			assert.NotContains(t, err.Error(), "top-secret")
			assert.Equal(t, 1, requestCount, "the login page must not be requested or parsed")
		})
	}
}

func TestClientAllowsOtherSameOriginRedirects(t *testing.T) {
	for _, target := range []string{"/canonical", "/login-help"} {
		t.Run(target, func(t *testing.T) {
			var requestCount int
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestCount++
				if r.URL.Path == "/v/P9Jkq9" {
					http.Redirect(w, r, target, http.StatusFound)
					return
				}
				_, _ = w.Write(readFixture(t, "detail-complete.html"))
			}))
			t.Cleanup(server.Close)

			client := newClientForServer(t, server)
			detail, err := client.Detail(context.Background(), "P9Jkq9")
			require.NoError(t, err)
			require.NotNil(t, detail)
			assert.Equal(t, 2, requestCount)
		})
	}
}

func TestClientHomeMapsQueryAndHasNext(t *testing.T) {
	var captured *url.URL
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := *r.URL
		captured = &u
		_, _ = w.Write(readFixture(t, "list-pagination.html"))
	}))
	t.Cleanup(server.Close)

	client := newClientForServer(t, server)
	page, err := client.Home(context.Background(), javdbapi.HomeQuery{
		Type:   javdbapi.HomeTypeCensored,
		Filter: javdbapi.HomeFilterDownload,
		Sort:   javdbapi.HomeSortMagnetDate,
		Page:   2,
	})
	require.NoError(t, err)
	require.NotNil(t, captured)

	assert.Equal(t, "/censored", captured.Path)
	q := captured.Query()
	assert.Equal(t, "1", q.Get("vft"))
	assert.Equal(t, "2", q.Get("vst"))
	assert.Equal(t, "2", q.Get("page"))
	assert.Equal(t, "zh", q.Get("locale"))
	assert.True(t, page.HasNext)
	assert.Equal(t, 2, page.Number)
	assert.NotEmpty(t, page.Items)
}

func TestClientSearchUsesFixedLocaleAndCustomUserAgent(t *testing.T) {
	var gotUA, gotLocale string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		gotLocale = r.URL.Query().Get("locale")
		_, _ = w.Write(readFixture(t, "list.html"))
	}))
	t.Cleanup(server.Close)

	client := newClientForServer(t, server, func(cfg *javdbapi.ClientConfig) {
		cfg.UserAgent = "javdbapi-test-agent/1.0"
	})
	_, err := client.Search(context.Background(), javdbapi.SearchQuery{Keyword: "VR", Page: 1})
	require.NoError(t, err)
	assert.Equal(t, "javdbapi-test-agent/1.0", gotUA)
	assert.Equal(t, string(javdbapi.LocaleZH), gotLocale)
}

func TestClientMapsEmptyResult(t *testing.T) {
	client := newClientForServer(t, newFixtureServer(t, readFixture(t, "list-empty.html"), func(string) {}))
	_, err := client.Search(context.Background(), javdbapi.SearchQuery{Keyword: "nonexistent", Page: 1})
	require.ErrorIs(t, err, javdbapi.ErrEmptyResult)
}

func TestClientMapsNotFoundError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	client := newClientForServer(t, server)
	_, err := client.Detail(context.Background(), "P9Jkq9")
	require.ErrorIs(t, err, javdbapi.ErrNotFound)

	var httpErr *javdbapi.HTTPError
	require.ErrorAs(t, err, &httpErr)
	assert.Equal(t, http.StatusNotFound, httpErr.StatusCode)
}

func TestClientMapsRateLimitedError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	t.Cleanup(server.Close)

	client := newClientForServer(t, server)
	_, err := client.Detail(context.Background(), "P9Jkq9")
	require.ErrorIs(t, err, javdbapi.ErrRateLimited)
}

func TestClientErrorsDoNotLeakSearchKeyword(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	client := newClientForServer(t, server)
	_, err := client.Search(context.Background(), javdbapi.SearchQuery{Keyword: "top-secret-query", Page: 1})
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "top-secret-query")
}

func TestClientLogsParseWarningsWithoutRawHTML(t *testing.T) {
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))

	server := newDetailFixtureServer(t, func(string) {})
	client := newClientForServer(t, server, func(cfg *javdbapi.ClientConfig) {
		cfg.Logger = logger
	})

	_, err := client.Detail(context.Background(), "P9Jkq9")
	require.NoError(t, err)

	logged := logBuf.String()
	assert.Contains(t, logged, "magnets.size")
	assert.NotContains(t, logged, "<html")
	assert.NotContains(t, logged, "<!DOCTYPE")
}

func TestClientSearchSucceedsWithPartiallyInvalidListAndLogsItemWarning(t *testing.T) {
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))

	server := newFixtureServer(t, readFixture(t, "list-partially-invalid.html"), func(string) {})
	client := newClientForServer(t, server, func(cfg *javdbapi.ClientConfig) {
		cfg.Logger = logger
	})

	page, err := client.Search(context.Background(), javdbapi.SearchQuery{Keyword: "VR", Page: 1})
	require.NoError(t, err)
	assert.NotEmpty(t, page.Items)

	logged := logBuf.String()
	assert.Contains(t, logged, "items[1]")
	assert.NotContains(t, logged, "<html")
	assert.NotContains(t, logged, "<!DOCTYPE")
	assert.NotContains(t, logged, "<div")
}

func TestClientReviewsSucceedsWithPartiallyInvalidReviewsAndLogsItemWarning(t *testing.T) {
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))

	server := newFixtureServer(t, readFixture(t, "reviews-partially-invalid.html"), func(string) {})
	client := newClientForServer(t, server, func(cfg *javdbapi.ClientConfig) {
		cfg.Logger = logger
	})

	reviews, err := client.Reviews(context.Background(), "P9Jkq9")
	require.NoError(t, err)
	assert.NotEmpty(t, reviews)

	logged := logBuf.String()
	assert.Contains(t, logged, "reviews[1]")
	assert.NotContains(t, logged, "<html")
	assert.NotContains(t, logged, "<!DOCTYPE")
	assert.NotContains(t, logged, "<div")
}

func TestClientVideoURL(t *testing.T) {
	client, err := javdbapi.NewClient(javdbapi.ClientConfig{
		Retry:     javdbapi.RetryPolicy{Disabled: true},
		RateLimit: javdbapi.RateLimitPolicy{Disabled: true},
	})
	require.NoError(t, err)
	assert.Equal(t, "https://javdb.com/v/P9Jkq9?locale=zh", client.VideoURL("P9Jkq9"))
}

func TestClientRankingBuildsFixedRoute(t *testing.T) {
	var captured *url.URL
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := *r.URL
		captured = &u
		_, _ = w.Write(readFixture(t, "list.html"))
	}))
	t.Cleanup(server.Close)

	client := newClientForServer(t, server)
	_, err := client.Ranking(context.Background(), javdbapi.RankingQuery{
		Period: javdbapi.RankingPeriodWeekly,
		Type:   javdbapi.RankingTypeCensored,
		Page:   1,
	})
	require.NoError(t, err)
	require.NotNil(t, captured)
	assert.Equal(t, "/rankings/movies", captured.Path)
	assert.Equal(t, "weekly", captured.Query().Get("p"))
	assert.Equal(t, "censored", captured.Query().Get("t"))
}

func TestClientActorVideosJoinsFilters(t *testing.T) {
	var captured *url.URL
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := *r.URL
		captured = &u
		_, _ = w.Write(readFixture(t, "list.html"))
	}))
	t.Cleanup(server.Close)

	client := newClientForServer(t, server)
	_, err := client.ActorVideos(context.Background(), javdbapi.ActorVideosQuery{
		ActorID: "vd5z",
		Filters: []javdbapi.ActorFilter{javdbapi.ActorFilterPlayable, javdbapi.ActorFilterDownload},
		Page:    1,
	})
	require.NoError(t, err)
	require.NotNil(t, captured)
	assert.Equal(t, "/actors/vd5z", captured.Path)
	assert.Equal(t, "p,d", captured.Query().Get("t"))
	assert.Equal(t, strconv.Itoa(0), captured.Query().Get("sort_type"))
}

func TestClientMakerVideosBuildsSortType(t *testing.T) {
	var captured *url.URL
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := *r.URL
		captured = &u
		_, _ = w.Write(readFixture(t, "list.html"))
	}))
	t.Cleanup(server.Close)

	client := newClientForServer(t, server)
	_, err := client.MakerVideos(context.Background(), javdbapi.MakerVideosQuery{
		MakerID: "ybW",
		Filter:  javdbapi.MakerFilterPlayable,
		Sort:    javdbapi.ListSortRating,
		Page:    1,
	})
	require.NoError(t, err)
	require.NotNil(t, captured)
	assert.Equal(t, "/makers/ybW", captured.Path)
	assert.Equal(t, "playable", captured.Query().Get("f"))
	assert.Equal(t, "1", captured.Query().Get("sort_type"))
}

func TestClientRejectsInvalidQueryWithoutRequest(t *testing.T) {
	var requested bool
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { requested = true }))
	t.Cleanup(server.Close)

	client := newClientForServer(t, server)
	_, err := client.Search(context.Background(), javdbapi.SearchQuery{Keyword: "", Page: 1})
	require.Error(t, err)
	assert.False(t, requested)
}

func TestClientSearchRejectsEmptyKeyword(t *testing.T) {
	for name, keyword := range map[string]string{
		"empty":           "",
		"whitespace-only": "   ",
	} {
		t.Run(name, func(t *testing.T) {
			var requested bool
			server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { requested = true }))
			t.Cleanup(server.Close)

			client := newClientForServer(t, server)
			_, err := client.Search(context.Background(), javdbapi.SearchQuery{Keyword: keyword, Page: 1})
			require.ErrorIs(t, err, javdbapi.ErrInvalidQuery)
			assert.False(t, requested)
		})
	}
}

func TestNewClientRejectsUnsupportedBaseURLScheme(t *testing.T) {
	for name, baseURL := range map[string]string{
		"ftp scheme":  "ftp://javdb.com",
		"no scheme":   "javdb.com",
		"file scheme": "file:///etc/passwd",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := javdbapi.NewClient(javdbapi.ClientConfig{BaseURL: baseURL})
			require.ErrorIs(t, err, javdbapi.ErrInvalidConfig)
		})
	}
}

func TestNewClientAcceptsHTTPAndHTTPSBaseURL(t *testing.T) {
	for _, baseURL := range []string{"http://javdb.com", "https://javdb.com"} {
		t.Run(baseURL, func(t *testing.T) {
			_, err := javdbapi.NewClient(javdbapi.ClientConfig{BaseURL: baseURL})
			require.NoError(t, err)
		})
	}
}

func TestNewClientRejectsBaseURLUserInfo(t *testing.T) {
	_, err := javdbapi.NewClient(javdbapi.ClientConfig{BaseURL: "https://user:test-proxy-secret@javdb.com"})
	require.ErrorIs(t, err, javdbapi.ErrInvalidConfig)
	assert.NotContains(t, err.Error(), "test-proxy-secret")
}

func TestNewClientBaseURLParseErrorDoesNotLeakInput(t *testing.T) {
	_, err := javdbapi.NewClient(javdbapi.ClientConfig{BaseURL: "https://user:test-proxy-secret@javdb.com\x00"})
	require.ErrorIs(t, err, javdbapi.ErrInvalidConfig)
	assert.NotContains(t, err.Error(), "test-proxy-secret")
}

func TestNewClientProxyErrorsDoNotLeakCredentials(t *testing.T) {
	const secret = "test-proxy-secret"
	cases := map[string]string{
		"malformed url":      "socks5://user:" + secret + "@host\x00",
		"empty host":         "socks5://user:" + secret + "@",
		"unsupported scheme": "ftp://user:" + secret + "@proxy.example:21",
	}
	for name, proxyURL := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := javdbapi.NewClient(javdbapi.ClientConfig{
				HTTP: javdbapi.HTTPConfig{ProxyURL: proxyURL},
			})
			require.ErrorIs(t, err, javdbapi.ErrInvalidConfig)
			assert.NotContains(t, err.Error(), secret)
		})
	}
}
