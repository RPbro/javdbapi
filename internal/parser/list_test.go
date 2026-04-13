package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseList(t *testing.T) {
	htmlBytes, err := os.ReadFile(filepath.Join("..", "testdata", "list.html"))
	require.NoError(t, err)

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(htmlBytes)))
	require.NoError(t, err)

	items, err := ParseList(doc)
	require.NoError(t, err)
	require.Len(t, items, 1)

	assert.Equal(t, "/v/abc123", items[0].Path)
	assert.Equal(t, "ABC-123", items[0].Code)
	assert.Equal(t, "Sample Title", items[0].Title)
	assert.Equal(t, "https://img.example/cover.jpg", items[0].CoverURL)
	assert.Equal(t, time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), items[0].PublishedAt)
	assert.InEpsilon(t, 4.1, items[0].Score, 0.0001)
	assert.Equal(t, 22, items[0].ScoreCount)
	assert.True(t, items[0].HasSubtitle)
}

func TestParseListReturnsErrorOnBadItem(t *testing.T) {
	html := `<html><body>
<div class="item">
  <a class="box" href="/v/abc123"><img src="https://img.example/cover.jpg"/></a>
  <div class="video-title">ABC-123 Sample Title</div>
  <div class="score"><span class="value">4.1分, 由22人評價</span></div>
  <div class="meta">2024-01-02</div>
  <div class="tag is-warning">含中字磁鏈</div>
</div>
<div class="item">
  <a class="box" href="/v/bad"><img src="https://img.example/bad.jpg"/></a>
  <div class="video-title">ABC-999</div>
  <div class="score"><span class="value">4.1分, 由22人評價</span></div>
  <div class="meta">2024-01-02</div>
</div>
</body></html>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	require.NoError(t, err)

	_, err = ParseList(doc)
	require.Error(t, err)
}

func TestParseListReturnsErrorOnMalformedScore(t *testing.T) {
	html := `<html><body>
<div class="item">
  <a class="box" href="/v/abc123"><img src="https://img.example/cover.jpg"/></a>
  <div class="video-title">ABC-123 Sample Title</div>
  <div class="score"><span class="value">bad score text</span></div>
  <div class="meta">2024-01-02</div>
</div>
</body></html>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	require.NoError(t, err)

	_, err = ParseList(doc)
	require.Error(t, err)
}

func TestParseListReturnsErrorOnMissingScoreText(t *testing.T) {
	html := `<html><body>
<div class="item">
  <a class="box" href="/v/abc123"><img src="https://img.example/cover.jpg"/></a>
  <div class="video-title">ABC-123 Sample Title</div>
  <div class="meta">2024-01-02</div>
</div>
</body></html>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	require.NoError(t, err)

	_, err = ParseList(doc)
	require.Error(t, err)
}

func TestParseListReturnsErrorOnMissingDateText(t *testing.T) {
	html := `<html><body>
<div class="item">
  <a class="box" href="/v/abc123"><img src="https://img.example/cover.jpg"/></a>
  <div class="video-title">ABC-123 Sample Title</div>
  <div class="score"><span class="value">4.1分, 由22人評價</span></div>
</div>
</body></html>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	require.NoError(t, err)

	_, err = ParseList(doc)
	require.Error(t, err)
}
