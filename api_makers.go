package javdbapi

const (
	MakersFilterAll         = ""
	MakersFilterPlayable    = "playable"
	MakersFilterSingle      = "single"
	MakersFilterCanDownload = "download"
	MakersFilterHasZH       = "cnsub"
	MakersFilterHasPreview  = "preview"
)

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

func (a *APIMakers) WithDetails() *APIMakers {
	a.base.WithDetails()
	return a
}

func (a *APIMakers) WithReviews() *APIMakers {
	a.base.WithReviews()
	return a
}

func (a *APIMakers) WithRandom() *APIMakers {
	a.base.WithRandom()
	return a
}

func (a *APIMakers) WithDebug() *APIMakers {
	a.base.WithDebug()
	return a
}

func (a *APIMakers) SetPage(page int) *APIMakers {
	a.base.SetPage(page)
	return a
}

func (a *APIMakers) SetLimit(limit int) *APIMakers {
	a.base.SetLimit(limit)
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

func (a *APIMakers) SetFilterCanDownload() *APIMakers {
	a.Filter = MakersFilterCanDownload
	return a
}

func (a *APIMakers) SetFilterHasZH() *APIMakers {
	a.Filter = MakersFilterHasZH
	return a
}

func (a *APIMakers) SetFilterHasPreview() *APIMakers {
	a.Filter = MakersFilterHasPreview
	return a
}

func (a *APIMakers) Get() ([]*JavDB, error) {
	return a.base.Get(a)
}
