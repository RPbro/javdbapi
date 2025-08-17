package javdbapi

type APIMakers struct {
	base   *API
	Maker  string
	Filter string
}

func (c *Client) GetMakers() *APIMakers {
	return &APIMakers{
		base: &API{
			client: c,
		},
		// see https://javdb.com/makers
		Maker:  "7R", // S1 NO.1 STYLE
		Filter: MakersFilterAll,
	}
}

func (a *APIMakers) SetPage(page int) *APIMakers {
	a.base.SetPage(page)
	return a
}

func (a *APIMakers) SetLimit(limit int) *APIMakers {
	a.base.SetLimit(limit)
	return a
}

func (a *APIMakers) SetFilter(filter Filter) *APIMakers {
	a.base.SetFilter(filter)
	return a
}

func (a *APIMakers) SetMaker(maker string) *APIMakers {
	a.Maker = maker
	return a
}

func (a *APIMakers) SetFilterAll() *APIMakers {
	a.Filter = MakersFilterAll
	return a
}

func (a *APIMakers) SetFilterPlayable() *APIMakers {
	a.Filter = MakersFilterPlayable
	return a
}

func (a *APIMakers) SetFilterSingle() *APIMakers {
	a.Filter = MakersFilterSingle
	return a
}

func (a *APIMakers) SetFilterDownload() *APIMakers {
	a.Filter = MakersFilterDownload
	return a
}

func (a *APIMakers) SetFilterCNSub() *APIMakers {
	a.Filter = MakersFilterCNSub
	return a
}

func (a *APIMakers) SetFilterPreview() *APIMakers {
	a.Filter = MakersFilterPreview
	return a
}

func (a *APIMakers) Get() ([]*Item, error) {
	return a.base.Get(a)
}
