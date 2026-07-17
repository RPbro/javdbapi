package scrape

import (
	"fmt"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func ParseReviews(doc *goquery.Document) (Result[[]Review], error) {
	result := Result[[]Review]{Value: []Review{}}

	items := doc.Find(selectorReviewItem)
	if items.Length() == 0 {
		if doc.Find(selectorEmptyState).Length() > 0 || hasReviewsEmptyMessage(doc) {
			return result, nil
		}
		return result, fmt.Errorf("%w: reviews page has no recognizable review items", ErrParse)
	}

	items.Each(func(i int, item *goquery.Selection) {
		review, warnings, err := parseReviewItem(item)
		if err != nil {
			result.Warnings = append(result.Warnings, Warning{Field: fmt.Sprintf("reviews[%d]", i), Message: err.Error()})
			return
		}
		result.Value = append(result.Value, review)
		for _, w := range warnings {
			result.Warnings = append(result.Warnings, Warning{Field: fmt.Sprintf("reviews[%d].%s", i, w.Field), Message: w.Message})
		}
	})
	if len(result.Value) == 0 {
		return result, fmt.Errorf("%w: reviews page has no valid review items", ErrParse)
	}

	return result, nil
}

// hasReviewsEmptyMessage reports whether doc contains the site's actual
// no-reviews notice. It checks both the selector and its exact text so an
// unrelated message box sharing the same "article.message.video-panel"
// markup (e.g. a site notice or error banner) is never mistaken for an
// empty reviews page.
func hasReviewsEmptyMessage(doc *goquery.Document) bool {
	found := false
	doc.Find(selectorReviewsMessageBody).EachWithBreak(func(_ int, s *goquery.Selection) bool {
		if strings.Contains(strings.TrimSpace(s.Text()), reviewsEmptyMessageText) {
			found = true
			return false
		}
		return true
	})
	return found
}

func parseReviewItem(item *goquery.Selection) (Review, []Warning, error) {
	var warnings []Warning

	idAttr, ok := item.Attr("id")
	if !ok {
		return Review{}, nil, fmt.Errorf("%w: review item missing id", ErrParse)
	}
	id := strings.TrimPrefix(idAttr, "review-item-")
	if id == "" || id == idAttr {
		return Review{}, nil, fmt.Errorf("%w: unexpected review item id %q", ErrParse, idAttr)
	}

	content := strings.TrimSpace(item.Find(".content").Text())
	if content == "" {
		return Review{}, nil, fmt.Errorf("%w: review item missing content", ErrParse)
	}

	var publishedAt *time.Time
	dateText := strings.TrimSpace(item.Find(".time").First().Text())
	if dateText != "" {
		dt, err := time.Parse(listDateLayout, dateText)
		if err != nil {
			warnings = append(warnings, Warning{Field: "published_at", Message: err.Error()})
		} else {
			publishedAt = &dt
		}
	}

	// Author is whatever free-standing text remains once every known
	// control/metadata element is stripped from a clone of the item.
	authorSel := item.Clone()
	authorSel.Find(".report.is-pulled-right, .likes.is-pulled-right, .score-stars, .time, .likes-count, .content").Remove()
	author := strings.TrimSpace(authorSel.Text())

	var score *int
	if stars := item.Find(".score-stars i.icon-star"); stars.Length() > 0 {
		n := stars.Not(".gray").Length()
		score = &n
	}

	likes := 0
	if likesText := strings.TrimSpace(item.Find(".likes-count").Text()); likesText != "" {
		n, err := parseCount(likesText)
		if err != nil {
			warnings = append(warnings, Warning{Field: "likes", Message: err.Error()})
		} else {
			likes = n
		}
	}

	return Review{
		ID:          id,
		Author:      author,
		Score:       score,
		PublishedAt: publishedAt,
		Likes:       likes,
		Content:     content,
	}, warnings, nil
}
