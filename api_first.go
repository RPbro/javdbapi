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

func (a *APIFirst) WithReviews() *APIFirst {
	a.base.WithReviews()
	return a
}

func (a *APIFirst) WithDebug() *APIFirst {
	a.base.WithDebug()
	return a
}

func (a *APIFirst) SetRaw(raw string) *APIFirst {
	a.Raw = raw
	return a
}

func (a *APIFirst) First() (*JavDB, error) {
	return a.base.First(a.Raw)
}
