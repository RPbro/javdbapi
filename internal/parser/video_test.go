package parser

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDetail(t *testing.T) {
	htmlBytes, err := os.ReadFile(filepath.Join("..", "testdata", "video.html"))
	require.NoError(t, err)

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(htmlBytes)))
	require.NoError(t, err)

	pageURL, err := url.Parse("https://javdb.com/v/abc123")
	require.NoError(t, err)

	detail, err := ParseDetail(doc, pageURL)
	require.NoError(t, err)

	assert.Equal(t, "/v/abc123", detail.Path)
	assert.Equal(t, "ABC-123", detail.Code)
	assert.Equal(t, "Sample Title", detail.Title)
	assert.Equal(t, "https://img.example/detail.jpg", detail.CoverURL)
	assert.Equal(t, time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), detail.PublishedAt)
	assert.InEpsilon(t, 4.1, detail.Score, 0.0001)
	assert.Equal(t, 22, detail.ScoreCount)
	assert.Equal(t, "https://cdn.example/trailer.mp4", detail.PreviewURL)
	assert.True(t, detail.HasSubtitle)
	assert.Equal(t, []string{"Julia"}, detail.Actors)
	assert.Equal(t, []string{"Drama", "中文字幕"}, detail.Tags)
	assert.Equal(t, []string{"https://img.example/shot-1.jpg"}, detail.Screenshots)
	require.Len(t, detail.Magnets, 1)
	assert.Equal(t, "magnet:?xt=urn:btih:1234567890abcdef1234567890abcdef12345678", detail.Magnets[0])
}

func TestParseDetailReturnsErrorOnMissingCode(t *testing.T) {
	pageURL, err := url.Parse("https://javdb.com/v/abc123")
	require.NoError(t, err)

	html := `<html><body>
<div class="current-title">Sample Title</div>
<img class="video-cover" src="https://img.example/detail.jpg" />
<div class="panel-block"><strong>日期:</strong><span>2024-01-02</span></div>
<div class="panel-block"><strong>評分:</strong><span>4.1分, 由22人評價</span></div>
</body></html>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	require.NoError(t, err)

	_, err = ParseDetail(doc, pageURL)
	require.Error(t, err)
}

func TestParseDetailReturnsErrorOnMalformedDate(t *testing.T) {
	pageURL, err := url.Parse("https://javdb.com/v/abc123")
	require.NoError(t, err)

	html := `<html><body>
<div class="current-title">Sample Title</div>
<img class="video-cover" src="https://img.example/detail.jpg" />
<div class="panel-block"><strong>番號:</strong><span>ABC-123</span></div>
<div class="panel-block"><strong>日期:</strong><span>2024-99-99</span></div>
</body></html>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	require.NoError(t, err)

	_, err = ParseDetail(doc, pageURL)
	require.Error(t, err)
}

func TestParseDetailReturnsErrorOnMalformedScore(t *testing.T) {
	pageURL, err := url.Parse("https://javdb.com/v/abc123")
	require.NoError(t, err)

	html := `<html><body>
<div class="current-title">Sample Title</div>
<img class="video-cover" src="https://img.example/detail.jpg" />
<div class="panel-block"><strong>番號:</strong><span>ABC-123</span></div>
<div class="panel-block"><strong>評分:</strong><span>4.1分, 由xx人評價</span></div>
</body></html>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	require.NoError(t, err)

	_, err = ParseDetail(doc, pageURL)
	require.Error(t, err)
}

func TestParseDetailReturnsErrorOnMissingDatePanel(t *testing.T) {
	pageURL, err := url.Parse("https://javdb.com/v/abc123")
	require.NoError(t, err)

	html := `<html><body>
<div class="current-title">Sample Title</div>
<img class="video-cover" src="https://img.example/detail.jpg" />
<div class="panel-block"><strong>番號:</strong><span>ABC-123</span></div>
<div class="panel-block"><strong>評分:</strong><span>4.1分, 由22人評價</span></div>
</body></html>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	require.NoError(t, err)

	_, err = ParseDetail(doc, pageURL)
	require.Error(t, err)
}

func TestParseDetailAllowsMissingScorePanel(t *testing.T) {
	pageURL, err := url.Parse("https://javdb.com/v/abc123")
	require.NoError(t, err)

	html := `<html><body>
<div class="current-title">Sample Title</div>
<img class="video-cover" src="https://img.example/detail.jpg" />
<div class="panel-block"><strong>番號:</strong><span>ABC-123</span></div>
<div class="panel-block"><strong>日期:</strong><span>2024-01-02</span></div>
</body></html>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	require.NoError(t, err)

	detail, err := ParseDetail(doc, pageURL)
	require.NoError(t, err)
	require.NotNil(t, detail)
	assert.Equal(t, "ABC-123", detail.Code)
	assert.Equal(t, time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), detail.PublishedAt)
	assert.Zero(t, detail.Score)
	assert.Zero(t, detail.ScoreCount)
}

func TestParseReviews(t *testing.T) {
	htmlBytes, err := os.ReadFile(filepath.Join("..", "testdata", "reviews.html"))
	require.NoError(t, err)

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(htmlBytes)))
	require.NoError(t, err)

	reviews, err := ParseReviews(doc)
	require.NoError(t, err)

	assert.Equal(t, []string{"Solid review text"}, reviews)
}
