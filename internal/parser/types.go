package parser

import "time"

type Summary struct {
	Path        string
	Title       string
	Code        string
	CoverURL    string
	PublishedAt time.Time
	Score       float64
	ScoreCount  int
	HasSubtitle bool
}

type Detail struct {
	Path        string
	Title       string
	Code        string
	CoverURL    string
	PublishedAt time.Time
	Score       float64
	ScoreCount  int
	HasSubtitle bool
	PreviewURL  string
	Actors      []string
	Tags        []string
	Screenshots []string
	Magnets     []string
}
