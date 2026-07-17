package scrape

import (
	"fmt"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const listDateLayout = "2006-01-02"

func videoIDFromPath(href string) (string, error) {
	return refIDFromHref(href, "v")
}

func ParseList(doc *goquery.Document, page int) (Result[ListPage], error) {
	result := Result[ListPage]{Value: ListPage{Items: []Summary{}, Number: page}}

	items := doc.Find(selectorListItem)
	if items.Length() == 0 {
		if doc.Find(selectorEmptyState).Length() > 0 {
			return result, fmt.Errorf("%w: list page has no items", ErrEmpty)
		}
		return result, fmt.Errorf("%w: list page has no recognizable items", ErrParse)
	}

	items.Each(func(i int, item *goquery.Selection) {
		summary, warnings, err := parseListItem(item)
		if err != nil {
			result.Warnings = append(result.Warnings, Warning{Field: fmt.Sprintf("items[%d]", i), Message: err.Error()})
			return
		}
		result.Value.Items = append(result.Value.Items, summary)
		for _, w := range warnings {
			result.Warnings = append(result.Warnings, Warning{Field: fmt.Sprintf("items[%d].%s", i, w.Field), Message: w.Message})
		}
	})
	if len(result.Value.Items) == 0 {
		return result, fmt.Errorf("%w: list page has no valid items", ErrParse)
	}

	result.Value.HasNext = doc.Find(selectorNextPage).Length() > 0
	return result, nil
}

func parseListItem(item *goquery.Selection) (Summary, []Warning, error) {
	var warnings []Warning

	href, ok := item.Find(".box").Attr("href")
	if !ok {
		return Summary{}, nil, fmt.Errorf("%w: list item missing box link", ErrParse)
	}
	id, err := videoIDFromPath(href)
	if err != nil {
		return Summary{}, nil, err
	}

	titleSel := item.Find(".video-title")
	code := strings.TrimSpace(titleSel.Find("strong").Text())
	fullTitle := strings.TrimSpace(titleSel.Text())
	title := strings.TrimSpace(strings.TrimPrefix(fullTitle, code))
	if code == "" || title == "" {
		return Summary{}, nil, fmt.Errorf("%w: list item missing code or title", ErrParse)
	}

	dateText := strings.TrimSpace(item.Find(".meta").Text())
	if dateText == "" {
		return Summary{}, nil, fmt.Errorf("%w: list item missing date", ErrParse)
	}
	publishedAt, err := time.Parse(listDateLayout, dateText)
	if err != nil {
		return Summary{}, nil, fmt.Errorf("%w: invalid list item date %q: %w", ErrParse, dateText, err)
	}

	scoreText := strings.TrimSpace(item.Find(".score").Text())
	var score *Score
	if scoreText != "" {
		if s, err := parseScore(scoreText); err != nil {
			warnings = append(warnings, Warning{Field: "score", Message: err.Error()})
		} else {
			score = &s
		}
	}

	coverURL, _ := item.Find(".cover img").Attr("src")

	tagText := item.Find(".tags").Text()
	hasSubtitleMagnet := strings.Contains(tagText, "含中字磁鏈")
	hasMagnet := hasSubtitleMagnet || strings.Contains(tagText, "含磁鏈")
	isPlayable := item.Find(".cover .tag-can-play").Length() > 0

	return Summary{
		ID:          id,
		Code:        code,
		Title:       title,
		CoverURL:    strings.TrimSpace(coverURL),
		PublishedAt: publishedAt,
		Score:       score,
		Availability: Availability{
			HasMagnet:         hasMagnet,
			HasSubtitleMagnet: hasSubtitleMagnet,
			IsPlayable:        isPlayable,
		},
	}, warnings, nil
}
