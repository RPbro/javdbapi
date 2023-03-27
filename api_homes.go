package javdbapi

const (
	HomesCategoryAll         = ""
	HomesCategoryCensored    = "censored"
	HomesCategoryUncensored  = "uncensored"
	HomesCategoryWestern     = "western"
	HomesSortByPubDate       = "1"
	HomesSortByMagnetUpdate  = "2"
	HomesFilterByAll         = "0"
	HomesFilterByCanDownload = "1"
	HomesFilterByHasZH       = "2"
	HomesFilterByHasReviews  = "3"
)

type APIHomes struct {
	base     *API
	Category string
	SortBy   string
	FilterBy string
}

func (c *Client) GetHomes() *APIHomes {
	return &APIHomes{
		base: &API{
			client: c,
		},
		Category: HomesCategoryAll,
		SortBy:   HomesSortByMagnetUpdate,
		FilterBy: HomesFilterByAll,
	}
}

func (a *APIHomes) WithDetails() *APIHomes {
	a.base.WithDetails()
	return a
}

func (a *APIHomes) WithReviews() *APIHomes {
	a.base.WithReviews()
	return a
}

func (a *APIHomes) WithRandom() *APIHomes {
	a.base.WithRandom()
	return a
}

func (a *APIHomes) WithDebug() *APIHomes {
	a.base.WithDebug()
	return a
}

func (a *APIHomes) SetPage(page int) *APIHomes {
	a.base.SetPage(page)
	return a
}

func (a *APIHomes) SetLimit(limit int) *APIHomes {
	a.base.SetLimit(limit)
	return a
}

func (a *APIHomes) SetFilter(filter Filter) *APIHomes {
	a.base.SetFilter(filter)
	return a
}

func (a *APIHomes) SetCategoryAll() *APIHomes {
	a.Category = HomesCategoryAll
	return a
}

func (a *APIHomes) SetCategoryCensored() *APIHomes {
	a.Category = HomesCategoryCensored
	return a
}

func (a *APIHomes) SetCategoryUncensored() *APIHomes {
	a.Category = HomesCategoryUncensored
	return a
}

func (a *APIHomes) SetCategoryWestern() *APIHomes {
	a.Category = HomesCategoryWestern
	return a
}

func (a *APIHomes) SetSortByPubDate() *APIHomes {
	a.SortBy = HomesSortByPubDate
	return a
}

func (a *APIHomes) SetSortByMagnetUpdate() *APIHomes {
	a.SortBy = HomesSortByMagnetUpdate
	return a
}

func (a *APIHomes) SetFilterByAll() *APIHomes {
	a.FilterBy = HomesFilterByAll
	return a
}

func (a *APIHomes) SetFilterByCanDownload() *APIHomes {
	a.FilterBy = HomesFilterByCanDownload
	return a
}

func (a *APIHomes) SetFilterByHasZH() *APIHomes {
	a.FilterBy = HomesFilterByHasZH
	return a
}

func (a *APIHomes) SetFilterByHasReviews() *APIHomes {
	a.FilterBy = HomesFilterByHasReviews
	return a
}

func (a *APIHomes) Get() ([]*JavDB, error) {
	return a.base.Get(a)
}
