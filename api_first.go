package javdbapi

type APIFirst struct {
	base *API
	Raw  string
}

func (c *Client) GetFirst() *APIFirst {
	return &APIFirst{
		base: &API{
			client: c,
		},
	}
}

func (a *APIFirst) SetDebug() *APIFirst {
	a.base.SetDebug()
	return a
}

func (a *APIFirst) SetRaw(raw string) *APIFirst {
	a.Raw = raw
	return a
}

func (a *APIFirst) First() (*Item, error) {
	return a.base.First(a.Raw)
}
