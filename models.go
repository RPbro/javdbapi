package javdbapi

import (
	"encoding/json"
	"time"
)

type Score struct {
	Value float64 `json:"value"`
	Count int     `json:"count"`
}

type Availability struct {
	HasMagnet         bool `json:"has_magnet"`
	HasSubtitleMagnet bool `json:"has_subtitle_magnet"`
	IsPlayable        bool `json:"is_playable"`
}

type NamedRef[ID ~string] struct {
	ID   ID     `json:"id"`
	Name string `json:"name"`
}

type VideoSummary struct {
	ID           VideoID      `json:"id"`
	Code         string       `json:"code"`
	Title        string       `json:"title"`
	CoverURL     string       `json:"cover_url,omitempty"`
	PublishedAt  time.Time    `json:"published_at"`
	Score        *Score       `json:"score,omitempty"`
	Availability Availability `json:"availability"`
}

type Gender string

const (
	GenderUnknown Gender = "unknown"
	GenderFemale  Gender = "female"
	GenderMale    Gender = "male"
)

type Actor struct {
	ID     ActorID `json:"id"`
	Name   string  `json:"name"`
	Gender Gender  `json:"gender"`
}

// Tag.Path keeps the full site path because tag query keys (c1, c2, c3, ...)
// are not a single stable global ID.
type Tag struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type Screenshot struct {
	FullURL      string `json:"full_url"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
}

type Magnet struct {
	URI         string     `json:"uri"`
	Name        string     `json:"name,omitempty"`
	SizeText    string     `json:"size_text,omitempty"`
	SizeBytes   *int64     `json:"size_bytes,omitempty"`
	FileCount   *int       `json:"file_count,omitempty"`
	HasSubtitle bool       `json:"has_subtitle"`
	IsHD        bool       `json:"is_hd"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
}

type VideoDetail struct {
	Summary         VideoSummary          `json:"summary"`
	DurationMinutes *int                  `json:"duration_minutes,omitempty"`
	Director        *NamedRef[DirectorID] `json:"director,omitempty"`
	Maker           *NamedRef[MakerID]    `json:"maker,omitempty"`
	Series          *NamedRef[SeriesID]   `json:"series,omitempty"`
	Actors          []Actor               `json:"actors"`
	Tags            []Tag                 `json:"tags"`
	Screenshots     []Screenshot          `json:"screenshots"`
	Magnets         []Magnet              `json:"magnets"`
	WantCount       *int                  `json:"want_count,omitempty"`
	WatchedCount    *int                  `json:"watched_count,omitempty"`
}

func NewVideoDetail(summary VideoSummary) VideoDetail {
	return VideoDetail{
		Summary:     summary,
		Actors:      []Actor{},
		Tags:        []Tag{},
		Screenshots: []Screenshot{},
		Magnets:     []Magnet{},
	}
}

// MarshalJSON guards against nil slices so JSON always encodes "[]", not "null",
// even if a VideoDetail is built without NewVideoDetail.
func (d VideoDetail) MarshalJSON() ([]byte, error) {
	type alias VideoDetail
	a := alias(d)
	if a.Actors == nil {
		a.Actors = []Actor{}
	}
	if a.Tags == nil {
		a.Tags = []Tag{}
	}
	if a.Screenshots == nil {
		a.Screenshots = []Screenshot{}
	}
	if a.Magnets == nil {
		a.Magnets = []Magnet{}
	}
	return json.Marshal(a)
}

type ReviewID string

type Review struct {
	ID          ReviewID   `json:"id"`
	Author      string     `json:"author,omitempty"`
	Score       *int       `json:"score,omitempty"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	Likes       int        `json:"likes"`
	Content     string     `json:"content"`
}
