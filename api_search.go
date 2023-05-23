package javdbapi

type APISearch struct {
	base  *API
	Query string
}

func (c *Client) GetSearch() *APISearch {
	return &APISearch{
		base: &API{
			client: c,
		},
	}
}

func (a *APISearch) WithDetails() *APISearch {
	a.base.WithDetails()
	return a
}

func (a *APISearch) WithReviews() *APISearch {
	a.base.WithReviews()
	return a
}

func (a *APISearch) WithDebug() *APISearch {
	a.base.WithDebug()
	return a
}

func (a *APISearch) SetPage(page int) *APISearch {
	a.base.SetPage(page)
	return a
}

func (a *APISearch) SetLimit(limit int) *APISearch {
	a.base.SetLimit(limit)
	return a
}

func (a *APISearch) SetFilter(filter Filter) *APISearch {
	a.base.SetFilter(filter)
	return a
}

func (a *APISearch) SetQuery(query string) *APISearch {
	a.Query = query
	return a
}

func (a *APISearch) Get() ([]*JavDB, error) {
	return a.base.Get(a)
}
