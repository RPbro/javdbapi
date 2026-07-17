package scrape

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDetailComplete(t *testing.T) {
	got, err := ParseDetail(loadFixture(t, "detail-complete.html"), "P9Jkq9")
	require.NoError(t, err)
	assert.Equal(t, "SNOS-177", got.Value.Summary.Code)
	assert.Equal(t, "P9Jkq9", got.Value.Summary.ID)
	require.NotNil(t, got.Value.DurationMinutes)
	assert.Equal(t, 120, *got.Value.DurationMinutes)
	require.NotNil(t, got.Value.Maker)
	assert.Equal(t, "7R", got.Value.Maker.ID)
	require.NotNil(t, got.Value.Director)
	assert.Equal(t, "DIR1", got.Value.Director.ID)
	require.NotNil(t, got.Value.Series)
	assert.Equal(t, "SER1", got.Value.Series.ID)
	require.Len(t, got.Value.Actors, 2)
	assert.Equal(t, GenderFemale, got.Value.Actors[0].Gender)
	assert.Equal(t, GenderMale, got.Value.Actors[1].Gender)
	assert.Equal(t, "vd5z", got.Value.Actors[0].ID)
	require.Len(t, got.Value.Tags, 2)
	assert.Equal(t, "/tags?c1=1", got.Value.Tags[0].Path)
	require.Len(t, got.Value.Screenshots, 2)
	assert.Equal(t, "https://example.com/screenshots/1-full.jpg", got.Value.Screenshots[0].FullURL)
	assert.Equal(t, "https://example.com/screenshots/1-thumb.jpg", got.Value.Screenshots[0].ThumbnailURL)
	require.NotNil(t, got.Value.WantCount)
	assert.Equal(t, 1234, *got.Value.WantCount)
	require.NotNil(t, got.Value.WatchedCount)
	assert.Equal(t, 567, *got.Value.WatchedCount)
	assert.True(t, got.Value.Summary.Availability.IsPlayable)

	require.NotEmpty(t, got.Value.Magnets)
	assert.Equal(t, "5.00GB", got.Value.Magnets[0].SizeText)
	assert.True(t, got.Value.Magnets[0].IsHD)
	assert.True(t, got.Value.Magnets[0].HasSubtitle)
	assert.Equal(t, "SNOS-177-C.mp4", got.Value.Magnets[0].Name, "name must come from .magnet-name .name, not the whole anchor text")
	assert.Equal(t, "magnet:?xt=urn:btih:AAAA1111&dn=SNOS-177-C.mp4", got.Value.Magnets[0].URI, "URI must prefer the copy button's data-clipboard-text over the anchor href")
	require.NotNil(t, got.Value.Magnets[0].PublishedAt)
	assert.Equal(t, "2024-01-05", got.Value.Magnets[0].PublishedAt.Format("2006-01-02"))
}

func TestParseDetailWarnsOnMalformedMagnetSize(t *testing.T) {
	got, err := ParseDetail(loadFixture(t, "detail-complete.html"), "P9Jkq9")
	require.NoError(t, err)
	require.Len(t, got.Value.Magnets, 2)
	assert.Nil(t, got.Value.Magnets[1].SizeBytes)
	assert.Equal(t, "magnet:?xt=urn:btih:BBBB2222", got.Value.Magnets[1].URI, "URI must fall back to the anchor href when no copy button is present")

	var found bool
	for _, w := range got.Warnings {
		if w.Field == "magnets.size" {
			found = true
		}
	}
	assert.True(t, found, "expected a magnets.size warning")
}

func TestParseDetailAllowsMissingOptionalSections(t *testing.T) {
	got, err := ParseDetail(loadFixture(t, "detail-minimal.html"), "OMqx0")
	require.NoError(t, err)
	assert.Empty(t, got.Value.Actors)
	assert.Empty(t, got.Value.Magnets)
	assert.Empty(t, got.Value.Tags)
	assert.Empty(t, got.Value.Screenshots)
	assert.Nil(t, got.Value.Series)
	assert.Nil(t, got.Value.Director)
	assert.Nil(t, got.Value.Maker)
	assert.Nil(t, got.Value.DurationMinutes)
	assert.Nil(t, got.Value.WantCount)
	assert.Nil(t, got.Value.WatchedCount)
}

func TestParseDetailWarnsOnEmptyScoreValue(t *testing.T) {
	got, err := ParseDetail(loadFixture(t, "detail-invalid.html"), "BAD000")
	require.NoError(t, err)
	assert.Nil(t, got.Value.Summary.Score)

	var found bool
	for _, w := range got.Warnings {
		if w.Field == "score" {
			found = true
		}
	}
	assert.True(t, found, "expected a score warning")
}

func TestParseDetailRejectsEmptyID(t *testing.T) {
	_, err := ParseDetail(loadFixture(t, "detail-minimal.html"), "  ")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrParse))
}

func TestParseDetailAllowsMissingScore(t *testing.T) {
	got, err := ParseDetail(loadFixture(t, "detail-no-score.html"), "NOSC000")
	require.NoError(t, err)
	assert.Nil(t, got.Value.Summary.Score)
	for _, w := range got.Warnings {
		assert.NotEqual(t, "score", w.Field, "missing score must not produce a warning")
	}
}

func TestParseDetailWarnsOnInvalidScore(t *testing.T) {
	got, err := ParseDetail(loadFixture(t, "detail-invalid-score.html"), "BADSC000")
	require.NoError(t, err)
	assert.Nil(t, got.Value.Summary.Score)

	var found bool
	for _, w := range got.Warnings {
		if w.Field == "score" {
			found = true
		}
	}
	assert.True(t, found, "expected a score warning")
}
