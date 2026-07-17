package scrape

import (
	"errors"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseReviews(t *testing.T) {
	got, err := ParseReviews(loadFixture(t, "reviews.html"))
	require.NoError(t, err)
	require.Len(t, got.Value, 1)
	assert.Equal(t, "218256471", got.Value[0].ID)
	assert.Equal(t, "wo***2", got.Value[0].Author)
	require.NotNil(t, got.Value[0].Score)
	assert.Equal(t, 4, *got.Value[0].Score, "score must count non-gray i.icon-star only")
	assert.Equal(t, 186, got.Value[0].Likes)
	require.NotNil(t, got.Value[0].PublishedAt)
	assert.Equal(t, "2026-03-10", got.Value[0].PublishedAt.Format("2006-01-02"))
	assert.Equal(t, "Great video, highly recommended.", got.Value[0].Content)
}

func TestParseReviewsEmpty(t *testing.T) {
	got, err := ParseReviews(loadFixture(t, "reviews-empty.html"))
	require.NoError(t, err)
	assert.Empty(t, got.Value)
	assert.NotNil(t, got.Value)
}

func TestParseReviewsRecognizesRealEmptyMessageBody(t *testing.T) {
	got, err := ParseReviews(loadFixture(t, "reviews-empty-message-body.html"))
	require.NoError(t, err)
	assert.NotNil(t, got.Value)
	assert.Empty(t, got.Value)
}

func TestParseReviewsUnrelatedMessageBodyStillErrors(t *testing.T) {
	_, err := ParseReviews(loadFixture(t, "reviews-unrelated-message-body.html"))
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrParse))
}

func TestParseReviewsRejectsUnrecognizedDocument(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader("<html><body><div class=\"unrelated\"></div></body></html>"))
	require.NoError(t, err)
	_, err = ParseReviews(doc)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrParse))
}

func TestParseReviewsSkipsMalformedItems(t *testing.T) {
	got, err := ParseReviews(loadFixture(t, "reviews-partially-invalid.html"))
	require.NoError(t, err)
	require.Len(t, got.Value, 5)

	ids := make([]string, len(got.Value))
	for i, r := range got.Value {
		ids[i] = r.ID
	}
	assert.Equal(t, []string{"1", "3", "4", "5", "6"}, ids)

	var skipWarning bool
	for _, w := range got.Warnings {
		if w.Field == "reviews[1]" {
			skipWarning = true
		}
	}
	assert.True(t, skipWarning, "expected a warning naming the skipped review index")
}

func TestParseReviewsFailsWhenAllItemsAreMalformed(t *testing.T) {
	_, err := ParseReviews(loadFixture(t, "reviews-all-invalid.html"))
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrParse))
}

func TestParseReviewAllowsMissingOptionalFields(t *testing.T) {
	got, err := ParseReviews(loadFixture(t, "reviews-partially-invalid.html"))
	require.NoError(t, err)

	byID := make(map[string]Review, len(got.Value))
	for _, r := range got.Value {
		byID[r.ID] = r
	}

	require.Contains(t, byID, "3")
	assert.Nil(t, byID["3"].PublishedAt, "missing date must be nil, not an error")

	require.Contains(t, byID, "5")
	assert.Empty(t, byID["5"].Author, "missing author must be empty, not an error")
}

func TestParseReviewWarnsOnMalformedOptionalFields(t *testing.T) {
	got, err := ParseReviews(loadFixture(t, "reviews-partially-invalid.html"))
	require.NoError(t, err)

	byID := make(map[string]Review, len(got.Value))
	for _, r := range got.Value {
		byID[r.ID] = r
	}

	require.Contains(t, byID, "4")
	assert.Nil(t, byID["4"].PublishedAt)
	require.Contains(t, byID, "6")
	assert.Equal(t, 0, byID["6"].Likes)

	var dateWarning, likesWarning bool
	for _, w := range got.Warnings {
		if w.Field == "reviews[3].published_at" {
			dateWarning = true
		}
		if w.Field == "reviews[5].likes" {
			likesWarning = true
		}
	}
	assert.True(t, dateWarning, "expected a published_at warning")
	assert.True(t, likesWarning, "expected a likes warning")
}
