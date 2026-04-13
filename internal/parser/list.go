package parser

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func ParseList(doc *goquery.Document) ([]Summary, error) {
	if doc == nil {
		return nil, fmt.Errorf("parse list: nil document")
	}

	var out []Summary
	var parseErr error

	doc.Find("div.item").EachWithBreak(func(_ int, selection *goquery.Selection) bool {
		titleParts := strings.Fields(selection.Find(".video-title").Text())
		if len(titleParts) < 2 {
			parseErr = fmt.Errorf("parse list: missing code/title")
			return false
		}

		code := strings.ToUpper(strings.TrimSpace(titleParts[0]))
		title := strings.TrimSpace(strings.Join(titleParts[1:], " "))
		if code == "" || title == "" {
			parseErr = fmt.Errorf("parse list: missing code/title")
			return false
		}

		path, ok := selection.Find(".box").Attr("href")
		if !ok || strings.TrimSpace(path) == "" {
			parseErr = fmt.Errorf("parse list: missing path")
			return false
		}
		path = strings.TrimSpace(path)

		coverURL, _ := selection.Find("img").Attr("src")
		coverURL = strings.TrimSpace(coverURL)

		scoreText := strings.TrimSpace(selection.Find(".score > .value").Text())
		if scoreText == "" {
			parseErr = fmt.Errorf("parse list: missing score")
			return false
		}
		score := 0.0
		scoreCount := 0
		scoreText = strings.ReplaceAll(scoreText, "由", "")
		scoreText = strings.ReplaceAll(scoreText, "人評價", "")
		parts := strings.Split(scoreText, ",")
		if len(parts) != 2 {
			parseErr = fmt.Errorf("parse list: invalid score text %q", scoreText)
			return false
		}
		scorePart := strings.TrimSuffix(strings.TrimSpace(parts[0]), "分")
		v, err := strconv.ParseFloat(strings.TrimSpace(scorePart), 64)
		if err != nil {
			parseErr = fmt.Errorf("parse list: invalid score %q: %w", scorePart, err)
			return false
		}
		score = v

		countPart := strings.TrimSpace(parts[1])
		n, err := strconv.Atoi(countPart)
		if err != nil {
			parseErr = fmt.Errorf("parse list: invalid score count %q: %w", countPart, err)
			return false
		}
		scoreCount = n

		var publishedAt time.Time
		dateText := strings.TrimSpace(selection.Find(".meta").Text())
		if dateText == "" {
			parseErr = fmt.Errorf("parse list: missing date")
			return false
		}
		dt, err := time.Parse("2006-01-02", dateText)
		if err != nil {
			parseErr = fmt.Errorf("parse list: invalid date %q: %w", dateText, err)
			return false
		}
		publishedAt = dt

		out = append(out, Summary{
			Path:        path,
			Title:       title,
			Code:        code,
			CoverURL:    coverURL,
			PublishedAt: publishedAt,
			Score:       score,
			ScoreCount:  scoreCount,
			HasSubtitle: strings.Contains(selection.Find(".tag, .is-warning").Text(), "含中字磁鏈"),
		})

		return true
	})

	if parseErr != nil {
		return nil, parseErr
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("parse list: no items")
	}

	return out, nil
}
