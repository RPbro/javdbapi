package javdbapi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/RPbro/javdbapi/internal/parser"
	"github.com/RPbro/javdbapi/internal/web"
)

func (c *Client) Video(ctx context.Context, query VideoQuery) (*Video, error) {
	rawURL, err := c.videoURL(query)
	if err != nil {
		return nil, err
	}

	body, err := c.runner.Get(ctx, rawURL)
	if err != nil {
		var use *web.UnexpectedStatusError
		if errors.As(err, &use) {
			return nil, fmt.Errorf("%w: %w", ErrUnexpectedStatus, err)
		}
		return nil, fmt.Errorf("fetch detail page: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("parse detail document: %w", err)
	}

	pageURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse detail url: %w", err)
	}

	detail, err := parser.ParseDetail(doc, pageURL)
	if err != nil {
		return nil, fmt.Errorf("parse detail page: %w", err)
	}

	reviewsBaseURL := c.baseURL
	if strings.TrimSpace(query.URL) != "" {
		u, err := url.Parse(rawURL)
		if err != nil {
			return nil, fmt.Errorf("parse detail url: %w", err)
		}
		reviewsBaseURL = &url.URL{Scheme: u.Scheme, Host: u.Host}
	}

	reviewsURL, err := web.BuildURL(reviewsBaseURL, detail.Path+"/reviews/lastest", map[string]string{
		"locale": "zh",
	})
	if err != nil {
		return nil, fmt.Errorf("build reviews url: %w", err)
	}

	reviewsBody, err := c.runner.Get(ctx, reviewsURL)
	if err != nil {
		var use *web.UnexpectedStatusError
		if errors.As(err, &use) {
			return nil, fmt.Errorf("%w: %w", ErrUnexpectedStatus, err)
		}
		return nil, fmt.Errorf("fetch reviews page: %w", err)
	}

	reviewsDoc, err := goquery.NewDocumentFromReader(bytes.NewReader(reviewsBody))
	if err != nil {
		return nil, fmt.Errorf("parse reviews document: %w", err)
	}

	hasReviews := reviewsDoc.Find(".review-item > .content").Length() != 0
	reviewsMessage := strings.TrimSpace(reviewsDoc.Find("article.message.video-panel .message-body").First().Text())
	isEmptyReviews := strings.Contains(reviewsMessage, "暫無內容")
	if !hasReviews && !isEmptyReviews {
		return nil, fmt.Errorf("parse reviews page: unexpected document")
	}

	reviews := []string{}
	if hasReviews {
		parsedReviews, err := parser.ParseReviews(reviewsDoc)
		if err != nil {
			return nil, fmt.Errorf("parse reviews page: %w", err)
		}
		reviews = parsedReviews
	}

	return &Video{
		ID:          detail.Path,
		Title:       detail.Title,
		Code:        detail.Code,
		URL:         rawURL,
		CoverURL:    detail.CoverURL,
		PublishedAt: detail.PublishedAt,
		Score:       detail.Score,
		ScoreCount:  detail.ScoreCount,
		HasSubtitle: detail.HasSubtitle,
		PreviewURL:  detail.PreviewURL,
		Actors:      detail.Actors,
		Tags:        detail.Tags,
		Screenshots: detail.Screenshots,
		Magnets:     detail.Magnets,
		Reviews:     reviews,
	}, nil
}

func (c *Client) videoURL(query VideoQuery) (string, error) {
	if strings.TrimSpace(query.URL) != "" {
		return query.URL, nil
	}
	if strings.TrimSpace(query.Path) == "" {
		return "", fmt.Errorf("%w: missing video path", ErrInvalidQuery)
	}

	rawURL, err := web.BuildURL(c.baseURL, query.Path, map[string]string{
		"locale": "zh",
	})
	if err != nil {
		return "", fmt.Errorf("build video url: %w", err)
	}
	return rawURL, nil
}
