package siteurl_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RPbro/javdbapi/internal/siteurl"
)

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	require.NoError(t, err)
	return u
}

func TestVideoAndReviewsURLs(t *testing.T) {
	base := mustURL(t, "https://javdb.com/base")
	video, err := siteurl.Video(base, "P9Jkq9", "zh")
	require.NoError(t, err)
	assert.Equal(t, "https://javdb.com/base/v/P9Jkq9?locale=zh", video.String())
	reviews, err := siteurl.Reviews(base, "P9Jkq9", "zh")
	require.NoError(t, err)
	assert.Equal(t, "https://javdb.com/base/v/P9Jkq9/reviews/lastest?locale=zh", reviews.String())
}

func TestVideoRejectsHostileID(t *testing.T) {
	base := mustURL(t, "https://javdb.com")
	for _, id := range []string{"", "../admin", "/v/x", "a/b", "a?b=1", "a#f"} {
		_, err := siteurl.Video(base, id, "zh")
		require.Error(t, err, "expected error for id %q", id)
	}
}

func TestSameOriginRedirect(t *testing.T) {
	original := mustURL(t, "https://javdb.com/v/x")

	cases := []struct {
		name    string
		next    string
		wantErr bool
	}{
		{"same origin different path", "https://javdb.com/v/y", false},
		{"host case difference", "https://JAVDB.com/v/y", false},
		{"implicit and explicit default port are equal", "https://javdb.com:443/v/y", false},
		{"hostname change", "https://evil.example/v/x", true},
		{"scheme change", "http://javdb.com/v/x", true},
		{"effective port change", "https://javdb.com:8443/v/x", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := siteurl.SameOriginRedirect(mustURL(t, tc.next), []*http.Request{{URL: original}})
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSameOriginRedirectRejectsHTTPSDowngrade(t *testing.T) {
	err := siteurl.SameOriginRedirect(
		mustURL(t, "http://javdb.com/v/x"),
		[]*http.Request{{URL: mustURL(t, "https://javdb.com/v/x")}},
	)
	require.Error(t, err)
}

func TestSameOriginRedirectAllowsFirstRequest(t *testing.T) {
	err := siteurl.SameOriginRedirect(mustURL(t, "https://javdb.com/v/x"), nil)
	require.NoError(t, err)
}

func TestSameOriginRedirectRejectsTenthRedirect(t *testing.T) {
	via := make([]*http.Request, 10)
	for i := range via {
		via[i] = &http.Request{URL: mustURL(t, "https://javdb.com/v/x")}
	}
	err := siteurl.SameOriginRedirect(mustURL(t, "https://javdb.com/v/y"), via)
	require.Error(t, err)
}

func TestSearchURL(t *testing.T) {
	base := mustURL(t, "https://javdb.com")
	u, err := siteurl.Search(base, "VR", 1, "zh")
	require.NoError(t, err)
	assert.Equal(t, "https://javdb.com/search?f=all&locale=zh&page=1&q=VR", u.String())
}

func TestSearchRejectsEmptyKeyword(t *testing.T) {
	_, err := siteurl.Search(mustURL(t, "https://javdb.com"), "   ", 1, "zh")
	require.Error(t, err)
}

func TestHomeURLWithType(t *testing.T) {
	base := mustURL(t, "https://javdb.com")
	u, err := siteurl.Home(base, "censored", "1", "2", 3, "zh")
	require.NoError(t, err)
	assert.Equal(t, "https://javdb.com/censored?locale=zh&page=3&vft=1&vst=2", u.String())
}

func TestHomeURLWithoutType(t *testing.T) {
	base := mustURL(t, "https://javdb.com")
	u, err := siteurl.Home(base, "", "", "", 0, "zh")
	require.NoError(t, err)
	assert.Equal(t, "https://javdb.com?locale=zh&page=1", u.String())
}

func TestMakerVideosURL(t *testing.T) {
	base := mustURL(t, "https://javdb.com")
	u, err := siteurl.MakerVideos(base, "ybW", "playable", 2, 1, "zh")
	require.NoError(t, err)
	assert.Equal(t, "https://javdb.com/makers/ybW?f=playable&locale=zh&page=1&sort_type=2", u.String())
}

func TestActorVideosURLJoinsFilters(t *testing.T) {
	base := mustURL(t, "https://javdb.com")
	u, err := siteurl.ActorVideos(base, "vd5z", []string{"p", "d"}, 0, 1, "zh")
	require.NoError(t, err)
	assert.Equal(t, "https://javdb.com/actors/vd5z?locale=zh&page=1&sort_type=0&t=p%2Cd", u.String())
}

func TestRankingURL(t *testing.T) {
	base := mustURL(t, "https://javdb.com")
	u, err := siteurl.Ranking(base, "daily", "censored", 1, "zh")
	require.NoError(t, err)
	assert.Equal(t, "https://javdb.com/rankings/movies?locale=zh&p=daily&page=1&t=censored", u.String())
}
