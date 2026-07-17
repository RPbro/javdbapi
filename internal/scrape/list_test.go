package scrape

import (
	"errors"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseListUsesRelNextWithoutInventingTotalPages(t *testing.T) {
	doc := loadFixture(t, "list-pagination.html")
	got, err := ParseList(doc, 1)
	require.NoError(t, err)
	assert.Equal(t, 1, got.Value.Number)
	assert.True(t, got.Value.HasNext)
	require.NotEmpty(t, got.Value.Items)
}

func TestParseListMapsAvailability(t *testing.T) {
	got, err := ParseList(loadFixture(t, "list.html"), 1)
	require.NoError(t, err)
	require.Len(t, got.Value.Items, 2)
	assert.True(t, got.Value.Items[0].Availability.HasMagnet)
	assert.True(t, got.Value.Items[0].Availability.HasSubtitleMagnet)
	assert.False(t, got.Value.Items[1].Availability.HasMagnet)
	assert.False(t, got.Value.Items[1].Availability.HasSubtitleMagnet)
}

func TestParseListExtractsFields(t *testing.T) {
	got, err := ParseList(loadFixture(t, "list.html"), 3)
	require.NoError(t, err)
	assert.Equal(t, 3, got.Value.Number)
	assert.False(t, got.Value.HasNext)

	first := got.Value.Items[0]
	assert.Equal(t, "P9Jkq9", first.ID)
	assert.Equal(t, "SNOS-177", first.Code)
	assert.Equal(t, "Sample Title One", first.Title)
	assert.Equal(t, "https://example.com/covers/p9jkq9.jpg", first.CoverURL)
	assert.Equal(t, "2024-01-02", first.PublishedAt.Format("2006-01-02"))
	assert.InDelta(t, 4.55, first.Score.Value, 0.0001)
	assert.Equal(t, 3080, first.Score.Count)
	assert.True(t, first.Availability.IsPlayable)
	assert.False(t, got.Value.Items[1].Availability.IsPlayable)
}

func TestParseListReturnsEmptyForRecognizedEmptyPage(t *testing.T) {
	_, err := ParseList(loadFixture(t, "list-empty.html"), 1)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEmpty))
}

func TestParseListInitializesItemsSlice(t *testing.T) {
	got, err := ParseList(loadFixture(t, "list-empty.html"), 1)
	require.Error(t, err)
	assert.NotNil(t, got.Value.Items)
}

func TestParseListSkipsMalformedItems(t *testing.T) {
	got, err := ParseList(loadFixture(t, "list-partially-invalid.html"), 1)
	require.NoError(t, err)
	require.Len(t, got.Value.Items, 3)
	assert.Equal(t, "VALD01", got.Value.Items[0].ID)
	assert.Equal(t, "VALD03", got.Value.Items[1].ID)
	assert.Equal(t, "VALD04", got.Value.Items[2].ID)

	var skipWarning bool
	for _, w := range got.Warnings {
		if w.Field == "items[1]" {
			skipWarning = true
		}
	}
	assert.True(t, skipWarning, "expected a warning naming the skipped item index")
}

func TestParseListFailsWhenAllItemsAreMalformed(t *testing.T) {
	_, err := ParseList(loadFixture(t, "list-all-invalid.html"), 1)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrParse))
}

func TestParseListAllowsMissingScore(t *testing.T) {
	got, err := ParseList(loadFixture(t, "list-partially-invalid.html"), 1)
	require.NoError(t, err)
	require.Len(t, got.Value.Items, 3)

	assert.Nil(t, got.Value.Items[1].Score, "item with no score element must be nil, not zero")

	assert.Nil(t, got.Value.Items[2].Score, "item with unparseable score text must be nil")
	var scoreWarning bool
	for _, w := range got.Warnings {
		if w.Field == "items[3].score" {
			scoreWarning = true
		}
	}
	assert.True(t, scoreWarning, "expected a score warning naming the malformed item index")
}

func TestParseListRejectsUnrecognizedEmptyPage(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader("<html><body><div class=\"unrelated\"></div></body></html>"))
	require.NoError(t, err)
	_, err = ParseList(doc, 1)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrParse))
	assert.False(t, errors.Is(err, ErrEmpty))
}
