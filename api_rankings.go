package javdbapi

type APIRankings struct {
	base   *API
	Period string
	Type   string
}

func (c *Client) GetRankings() *APIRankings {
	return &APIRankings{
		base: &API{
			client: c,
		},
		Period: RankingsPeriodDaily,
		Type:   RankingsTypeCensored,
	}
}

func (a *APIRankings) SetDebug() *APIRankings {
	a.base.SetDebug()
	return a
}

func (a *APIRankings) SetRandom() *APIRankings {
	a.base.SetRandom()
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

func (a *APIRankings) SetPeriodDaily() *APIRankings {
	a.Period = RankingsPeriodDaily
	return a
}

func (a *APIRankings) SetPeriodWeekly() *APIRankings {
	a.Period = RankingsPeriodWeekly
	return a
}

func (a *APIRankings) SetPeriodMonthly() *APIRankings {
	a.Period = RankingsPeriodMonthly
	return a
}

func (a *APIRankings) SetTypeCensored() *APIRankings {
	a.Type = RankingsTypeCensored
	return a
}

func (a *APIRankings) SetTypeUncensored() *APIRankings {
	a.Type = RankingsTypeUncensored
	return a
}

func (a *APIRankings) SetTypeWestern() *APIRankings {
	a.Type = RankingsTypeWestern
	return a
}

func (a *APIRankings) Get() ([]*Item, error) {
	return a.base.Get(a)
}
