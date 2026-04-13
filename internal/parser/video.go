package parser

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func ParseDetail(doc *goquery.Document, pageURL *url.URL) (*Detail, error) {
	if doc == nil {
		return nil, fmt.Errorf("parse detail: nil document")
	}
	if pageURL == nil {
		return nil, fmt.Errorf("parse detail: missing page url")
	}

	detail := &Detail{
		Path:  pageURL.Path,
		Title: strings.TrimSpace(doc.Find(".current-title").First().Text()),
	}

	if detail.Title == "" {
		return nil, fmt.Errorf("parse detail: missing title")
	}

	detail.CoverURL, _ = doc.Find(".video-cover").Attr("src")

	var parseErr error
	sawDate := false
	doc.Find(".panel-block").EachWithBreak(func(_ int, selection *goquery.Selection) bool {
		label := strings.TrimSpace(selection.Find("strong").First().Text())
		value := strings.TrimSpace(selection.Find("span").First().Text())

		switch label {
		case "番號:":
			detail.Code = strings.ToUpper(value)
		case "日期:":
			sawDate = true
			if value == "" {
				parseErr = fmt.Errorf("parse detail: missing date")
				return false
			}
			publishedAt, err := time.Parse("2006-01-02", value)
			if err != nil {
				parseErr = fmt.Errorf("parse detail: invalid date %q: %w", value, err)
				return false
			}
			detail.PublishedAt = publishedAt
		case "評分:":
			if value == "" {
				parseErr = fmt.Errorf("parse detail: missing score")
				return false
			}
			scoreText := strings.ReplaceAll(value, "由", "")
			scoreText = strings.ReplaceAll(scoreText, "人評價", "")
			parts := strings.Split(scoreText, ",")
			if len(parts) != 2 {
				parseErr = fmt.Errorf("parse detail: invalid score text %q", value)
				return false
			}

			scorePart := strings.TrimSuffix(strings.TrimSpace(parts[0]), "分")
			score, err := strconv.ParseFloat(strings.TrimSpace(scorePart), 64)
			if err != nil {
				parseErr = fmt.Errorf("parse detail: invalid score %q: %w", scorePart, err)
				return false
			}

			countPart := strings.TrimSpace(parts[1])
			count, err := strconv.Atoi(countPart)
			if err != nil {
				parseErr = fmt.Errorf("parse detail: invalid score count %q: %w", countPart, err)
				return false
			}
			detail.Score = score
			detail.ScoreCount = count
		case "演員:":
			selection.Find(".value > .female").Each(func(_ int, actorSelection *goquery.Selection) {
				actor := strings.TrimSpace(actorSelection.Prev().Text())
				if actor != "" {
					detail.Actors = append(detail.Actors, actor)
				}
			})
		case "類別:":
			selection.Find(".value > a").Each(func(_ int, tagSelection *goquery.Selection) {
				tag := strings.TrimSpace(tagSelection.Text())
				if tag != "" {
					detail.Tags = append(detail.Tags, tag)
				}
			})
		}
		return true
	})
	if parseErr != nil {
		return nil, parseErr
	}
	if strings.TrimSpace(detail.Code) == "" {
		return nil, fmt.Errorf("parse detail: missing code")
	}
	if !sawDate {
		return nil, fmt.Errorf("parse detail: missing date")
	}

	if previewURL, ok := doc.Find("#preview-video source").Attr("src"); ok {
		if strings.HasPrefix(previewURL, "//") {
			previewURL = "https:" + previewURL
		}
		detail.PreviewURL = previewURL
	}

	detail.HasSubtitle = strings.Contains(doc.Find(".tag, .is-warning").Text(), "含中字磁鏈")

	doc.Find(".preview-images .tile-item").Each(func(_ int, selection *goquery.Selection) {
		if screenshot, ok := selection.Attr("href"); ok && strings.TrimSpace(screenshot) != "" {
			detail.Screenshots = append(detail.Screenshots, screenshot)
		}
	})

	doc.Find("#magnets-content .magnet-name > a").Each(func(_ int, selection *goquery.Selection) {
		if magnet, ok := selection.Attr("href"); ok && strings.HasPrefix(magnet, "magnet:?xt=urn:btih:") {
			detail.Magnets = append(detail.Magnets, magnet)
		}
	})

	return detail, nil
}
