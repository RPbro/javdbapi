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

func (a *APIRaw) SetDebug() *APIRaw {
	a.base.SetDebug()
	return a
}

func (a *APIRaw) SetRandom() *APIRaw {
	a.base.SetRandom()
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

func (a *APIRaw) SetFilter(filter Filter) *APIRaw {
	a.base.SetFilter(filter)
	return a
}

func (a *APIRaw) SetRaw(raw string) *APIRaw {
	a.Raw = raw
	return a
}

func (a *APIRaw) Get() ([]*Item, error) {
	return a.base.Get(a)
}
