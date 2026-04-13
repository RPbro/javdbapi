package parser

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func ParseReviews(doc *goquery.Document) ([]string, error) {
	var reviews []string

	doc.Find(".review-item > .content").Each(func(_ int, selection *goquery.Selection) {
		review := strings.TrimSpace(selection.Text())
		if review != "" {
			reviews = append(reviews, review)
		}
	})

	if len(reviews) == 0 {
		return nil, fmt.Errorf("parse reviews: no items")
	}

	return reviews, nil
}
