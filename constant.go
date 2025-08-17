package javdbapi

import "time"

// client
const (
	defaultDomain    = "https://javdb.com"
	defaultTimeout   = time.Second * 30
	defaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36"
	defaultPage      = 1
	defaultPageMax   = 60
)

// path
const (
	PathHome     = "/"
	PathRankings = "/rankings/movies"
	PathMakers   = "/makers"
	PathActors   = "/actors"
	PathSearch   = "/search"
	PathReviews  = "/reviews/lastest"
)

// homes https://javdb.com/censored?vft=2
const (
	HomeTypeAll         = ""
	HomeTypeCensored    = "censored"
	HomeTypeUncensored  = "uncensored"
	HomeTypeWestern     = "western"
	HomeFilterAll       = "0"
	HomeFilterDownload  = "1"
	HomeFilterCNSub     = "2"
	HomeFilterReview    = "3"
	HomeSortPublishDate = "1"
	HomeSortMagnetDate  = "2"
)

// rankings https://javdb.com/rankings/movies?p=daily&t=censored
const (
	RankingsPeriodDaily    = "daily"
	RankingsPeriodWeekly   = "weekly"
	RankingsPeriodMonthly  = "monthly"
	RankingsTypeCensored   = "censored"
	RankingsTypeUncensored = "uncensored"
	RankingsTypeWestern    = "western"
)

// https://javdb.com/makers/7R?f=download
const (
	MakersFilterAll      = ""
	MakersFilterPlayable = "playable"
	MakersFilterSingle   = "single"
	MakersFilterDownload = "download"
	MakersFilterCNSub    = "cnsub"
	MakersFilterPreview  = "preview"
)

// actors https://javdb.com/actors/O2Q30
const (
	ActorsFilterAll      = ""
	ActorsFilterPlayable = "p"
	ActorsFilterSingle   = "s"
	ActorsFilterDownload = "d"
	ActorsFilterCNSub    = "c"
)
