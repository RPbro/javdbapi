package javdbapi

type APIHome struct {
	base   *API
	Type   string
	Filter string
	Sort   string
}

func (c *Client) GetHome() *APIHome {
	return &APIHome{
		base: &API{
			client: c,
		},
		Type:   HomeTypeAll,
		Filter: HomeFilterAll,
		Sort:   HomeSortMagnetDate,
	}
}

func (a *APIHome) SetDebug() *APIHome {
	a.base.SetDebug()
	return a
}

func (a *APIHome) SetRandom() *APIHome {
	a.base.SetRandom()
	return a
}

func (a *APIHome) SetPage(page int) *APIHome {
	a.base.SetPage(page)
	return a
}

func (a *APIHome) SetLimit(limit int) *APIHome {
	a.base.SetLimit(limit)
	return a
}

func (a *APIHome) SetFilter(filter Filter) *APIHome {
	a.base.SetFilter(filter)
	return a
}

func (a *APIHome) SetTypeAll() *APIHome {
	a.Type = HomeTypeAll
	return a
}

func (a *APIHome) SetTypeCensored() *APIHome {
	a.Type = HomeTypeCensored
	return a
}

func (a *APIHome) SetTypeUncensored() *APIHome {
	a.Type = HomeTypeUncensored
	return a
}

func (a *APIHome) SetTypeWestern() *APIHome {
	a.Type = HomeTypeWestern
	return a
}

func (a *APIHome) SetSortPublishDate() *APIHome {
	a.Sort = HomeSortPublishDate
	return a
}

func (a *APIHome) SetSortMagnetDate() *APIHome {
	a.Sort = HomeSortMagnetDate
	return a
}

func (a *APIHome) SetFilterAll() *APIHome {
	a.Filter = HomeFilterAll
	return a
}

func (a *APIHome) SetFilterDownload() *APIHome {
	a.Filter = HomeFilterDownload
	return a
}

func (a *APIHome) SetFilterCNSub() *APIHome {
	a.Filter = HomeFilterCNSub
	return a
}

func (a *APIHome) SetFilterReview() *APIHome {
	a.Filter = HomeFilterReview
	return a
}

func (a *APIHome) Get() ([]*Item, error) {
	return a.base.Get(a)
}
