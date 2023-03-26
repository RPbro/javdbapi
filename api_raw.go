package javdbapi

type APIRaw struct {
	base *API
	Raw  string
}

func (c *Client) GetRaw() *APIRaw {
	return &APIRaw{
		base: &API{
			client: c,
		},
	}
}

func (a *APIRaw) WithDetails() *APIRaw {
	a.base.WithDetails()
	return a
}

func (a *APIRaw) WithReviews() *APIRaw {
	a.base.WithReviews()
	return a
}

func (a *APIRaw) WithRandom() *APIRaw {
	a.base.WithRandom()
	return a
}

func (a *APIRaw) WithDebug() *APIRaw {
	a.base.WithDebug()
	return a
}

func (a *APIRaw) SetPage(page int) *APIRaw {
	a.base.SetPage(page)
	return a
}

func (a *APIRaw) SetLimit(limit int) *APIRaw {
	a.base.SetLimit(limit)
	return a
}

func (a *APIRaw) SetRaw(raw string) *APIRaw {
	a.Raw = raw
	return a
}

func (a *APIRaw) Get() ([]*JavDB, error) {
	return a.base.Get(a)
}
