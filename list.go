package javdbapi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/RPbro/javdbapi/internal/parser"
	"github.com/RPbro/javdbapi/internal/web"
)

func (c *Client) Home(ctx context.Context, query HomeQuery) ([]Video, error) {
	route := "/"
	if strings.TrimSpace(string(query.Type)) != "" {
		route = "/" + string(query.Type)
	}

	return c.fetchList(ctx, route, map[string]string{
		"vft":    string(query.Filter),
		"vst":    string(query.Sort),
		"page":   pageValue(query.Page),
		"locale": "zh",
	})
}

func (c *Client) Search(ctx context.Context, query SearchQuery) ([]Video, error) {
	if strings.TrimSpace(query.Keyword) == "" {
		return nil, fmt.Errorf("%w: missing keyword", ErrInvalidQuery)
	}

	return c.fetchList(ctx, "/search", map[string]string{
		"q":      query.Keyword,
		"f":      "all",
		"page":   pageValue(query.Page),
		"locale": "zh",
	})
}

func (c *Client) Maker(ctx context.Context, query MakerQuery) ([]Video, error) {
	if strings.TrimSpace(query.MakerID) == "" {
		return nil, fmt.Errorf("%w: missing maker id", ErrInvalidQuery)
	}

	return c.fetchList(ctx, "/makers/"+query.MakerID, map[string]string{
		"f":      string(query.Filter),
		"page":   pageValue(query.Page),
		"locale": "zh",
	})
}

func (c *Client) Actor(ctx context.Context, query ActorQuery) ([]Video, error) {
	if strings.TrimSpace(query.ActorID) == "" {
		return nil, fmt.Errorf("%w: missing actor id", ErrInvalidQuery)
	}

	values := make([]string, 0, len(query.Filters))
	for _, filter := range query.Filters {
		v := strings.TrimSpace(string(filter))
		if v != "" {
			values = append(values, v)
		}
	}

	return c.fetchList(ctx, "/actors/"+query.ActorID, map[string]string{
		"t":      strings.Join(values, ","),
		"page":   pageValue(query.Page),
		"locale": "zh",
	})
}

func (c *Client) Ranking(ctx context.Context, query RankingQuery) ([]Video, error) {
	return c.fetchList(ctx, "/rankings/movies", map[string]string{
		"p":      string(query.Period),
		"t":      string(query.Type),
		"page":   pageValue(query.Page),
		"locale": "zh",
	})
}

func (c *Client) fetchList(ctx context.Context, route string, params map[string]string) ([]Video, error) {
	rawURL, err := web.BuildURL(c.baseURL, route, params)
	if err != nil {
		return nil, fmt.Errorf("build list url: %w", err)
	}

	body, err := c.runner.Get(ctx, rawURL)
	if err != nil {
		var use *web.UnexpectedStatusError
		if errors.As(err, &use) {
			return nil, fmt.Errorf("%w: %w", ErrUnexpectedStatus, err)
		}
		return nil, fmt.Errorf("fetch list page: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("parse list document: %w", err)
	}

	emptyMessage := strings.TrimSpace(doc.Find(".empty-message").First().Text())
	if doc.Find("div.item").Length() == 0 && strings.Contains(emptyMessage, "暫無內容") {
		return nil, fmt.Errorf("%w: %s", ErrEmptyResult, rawURL)
	}

	summaries, err := parser.ParseList(doc)
	if err != nil {
		return nil, fmt.Errorf("parse list page: %w", err)
	}

	videos := make([]Video, 0, len(summaries))
	for _, summary := range summaries {
		videoURL, err := web.BuildURL(c.baseURL, summary.Path, map[string]string{
			"locale": "zh",
		})
		if err != nil {
			return nil, fmt.Errorf("build video url: %w", err)
		}

		videos = append(videos, Video{
			ID:          summary.Path,
			Title:       summary.Title,
			Code:        summary.Code,
			URL:         videoURL,
			CoverURL:    summary.CoverURL,
			PublishedAt: summary.PublishedAt,
			Score:       summary.Score,
			ScoreCount:  summary.ScoreCount,
			HasSubtitle: summary.HasSubtitle,
		})
	}

	if len(videos) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrEmptyResult, rawURL)
	}

	return videos, nil
}

func pageValue(page int) string {
	if page <= 0 {
		page = 1
	}
	return strconv.Itoa(page)
}
