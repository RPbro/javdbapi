package scrape

import "time"

type Score struct {
	Value float64
	Count int
}

type Availability struct {
	HasMagnet         bool
	HasSubtitleMagnet bool
	IsPlayable        bool
}

type Summary struct {
	ID           string
	Code         string
	Title        string
	CoverURL     string
	PublishedAt  time.Time
	Score        *Score
	Availability Availability
}

type ListPage struct {
	Items   []Summary
	Number  int
	HasNext bool
}

type Gender string

const (
	GenderUnknown Gender = "unknown"
	GenderFemale  Gender = "female"
	GenderMale    Gender = "male"
)

type Actor struct {
	ID     string
	Name   string
	Gender Gender
}

type Tag struct {
	Name string
	Path string
}

type Screenshot struct {
	FullURL      string
	ThumbnailURL string
}

type Magnet struct {
	URI         string
	Name        string
	SizeText    string
	SizeBytes   *int64
	FileCount   *int
	HasSubtitle bool
	IsHD        bool
	PublishedAt *time.Time
}

type NamedRef struct {
	ID   string
	Name string
}

type Detail struct {
	Summary         Summary
	DurationMinutes *int
	Director        *NamedRef
	Maker           *NamedRef
	Series          *NamedRef
	Actors          []Actor
	Tags            []Tag
	Screenshots     []Screenshot
	Magnets         []Magnet
	WantCount       *int
	WatchedCount    *int
}

type Review struct {
	ID          string
	Author      string
	Score       *int
	PublishedAt *time.Time
	Likes       int
	Content     string
}
