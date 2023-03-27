package javdbapi

const (
	RankingsCategoryCensored   = "censored"
	RankingsCategoryUncensored = "uncensored"
	RankingsCategoryWestern    = "western"
	RankingsTimeDaily          = "daily"
	RankingsTimeWeekly         = "weekly"
	RankingsTimeMonthly        = "monthly"
)

type APIRankings struct {
	base     *API
	Category string
	Time     string
}

func (c *Client) GetRankings() *APIRankings {
	return &APIRankings{
		base: &API{
			client: c,
		},
		Category: RankingsCategoryCensored,
		Time:     RankingsTimeDaily,
	}
}

func (a *APIRankings) WithDetails() *APIRankings {
	a.base.WithDetails()
	return a
}

func (a *APIRankings) WithReviews() *APIRankings {
	a.base.WithReviews()
	return a
}

func (a *APIRankings) WithRandom() *APIRankings {
	a.base.WithRandom()
	return a
}

func (a *APIRankings) WithDebug() *APIRankings {
	a.base.WithDebug()
	return a
}

func (a *APIRankings) SetPage(page int) *APIRankings {
	a.base.SetPage(page)
	return a
}

func (a *APIRankings) SetLimit(limit int) *APIRankings {
	a.base.SetLimit(limit)
	return a
}

func (a *APIRankings) SetFilter(filter Filter) *APIRankings {
	a.base.SetFilter(filter)
	return a
}

func (a *APIRankings) SetCategoryCensored() *APIRankings {
	a.Category = RankingsCategoryCensored
	return a
}

func (a *APIRankings) SetCategoryUncensored() *APIRankings {
	a.Category = RankingsCategoryUncensored
	return a
}

func (a *APIRankings) SetCategoryWestern() *APIRankings {
	a.Category = RankingsCategoryWestern
	return a
}

func (a *APIRankings) SetTimeDaily() *APIRankings {
	a.Time = RankingsTimeDaily
	return a
}

func (a *APIRankings) SetTimeWeekly() *APIRankings {
	a.Time = RankingsTimeWeekly
	return a
}

func (a *APIRankings) SetTimeMonthly() *APIRankings {
	a.Time = RankingsTimeMonthly
	return a
}

func (a *APIRankings) Get() ([]*JavDB, error) {
	return a.base.Get(a)
}
