package javdbapi

import "time"

type Config struct {
	BaseURL   string
	Timeout   time.Duration
	ProxyURL  string
	UserAgent string
	Debug     bool
}

type Video struct {
	ID          string
	Title       string
	Code        string
	URL         string
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
	Reviews     []string
}

type HomeType string
type HomeFilter string
type HomeSort string

const (
	HomeTypeAll        HomeType = ""
	HomeTypeCensored   HomeType = "censored"
	HomeTypeUncensored HomeType = "uncensored"
	HomeTypeWestern    HomeType = "western"
)

const (
	HomeFilterAll      HomeFilter = "0"
	HomeFilterDownload HomeFilter = "1"
	HomeFilterCNSub    HomeFilter = "2"
	HomeFilterReview   HomeFilter = "3"
)

const (
	HomeSortPublishDate HomeSort = "1"
	HomeSortMagnetDate  HomeSort = "2"
)

type HomeQuery struct {
	Type   HomeType
	Filter HomeFilter
	Sort   HomeSort
	Page   int
}

type SearchQuery struct {
	Keyword string
	Page    int
}

type MakerFilter string

const (
	MakerFilterAll      MakerFilter = ""
	MakerFilterPlayable MakerFilter = "playable"
	MakerFilterSingle   MakerFilter = "single"
	MakerFilterDownload MakerFilter = "download"
	MakerFilterCNSub    MakerFilter = "cnsub"
	MakerFilterPreview  MakerFilter = "preview"
)

type MakerQuery struct {
	MakerID string
	Filter  MakerFilter
	Page    int
}

type ActorFilter string

const (
	ActorFilterAll      ActorFilter = ""
	ActorFilterPlayable ActorFilter = "p"
	ActorFilterSingle   ActorFilter = "s"
	ActorFilterDownload ActorFilter = "d"
	ActorFilterCNSub    ActorFilter = "c"
)

type ActorQuery struct {
	ActorID string
	Filters []ActorFilter
	Page    int
}

type RankingPeriod string
type RankingType string

const (
	RankingPeriodDaily   RankingPeriod = "daily"
	RankingPeriodWeekly  RankingPeriod = "weekly"
	RankingPeriodMonthly RankingPeriod = "monthly"
)

const (
	RankingTypeCensored   RankingType = "censored"
	RankingTypeUncensored RankingType = "uncensored"
	RankingTypeWestern    RankingType = "western"
)

type RankingQuery struct {
	Period RankingPeriod
	Type   RankingType
	Page   int
}

type VideoQuery struct {
	URL  string
	Path string
}
