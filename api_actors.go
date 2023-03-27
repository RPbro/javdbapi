package javdbapi

const (
	ActorsFilterAll         = ""
	ActorsFilterPlayable    = "p"
	ActorsFilterSingle      = "s"
	ActorsFilterCanDownload = "d"
	ActorsFilterHasZH       = "c"
)

type APIActors struct {
	base   *API
	Actor  string
	Filter []string
}

func (c *Client) GetActors() *APIActors {
	return &APIActors{
		base: &API{
			client: c,
		},
		// see https://javdb.com/actors
		Actor:  "M4Q7", // 明里つむぎ
		Filter: []string{},
	}
}

func (a *APIActors) WithDetails() *APIActors {
	a.base.WithDetails()
	return a
}

func (a *APIActors) WithReviews() *APIActors {
	a.base.WithReviews()
	return a
}

func (a *APIActors) WithRandom() *APIActors {
	a.base.WithRandom()
	return a
}

func (a *APIActors) WithDebug() *APIActors {
	a.base.WithDebug()
	return a
}

func (a *APIActors) SetPage(page int) *APIActors {
	a.base.SetPage(page)
	return a
}

func (a *APIActors) SetLimit(limit int) *APIActors {
	a.base.SetLimit(limit)
	return a
}

func (a *APIActors) SetFilter(filter Filter) *APIActors {
	a.base.SetFilter(filter)
	return a
}

func (a *APIActors) SetActor(actor string) *APIActors {
	a.Actor = actor
	return a
}

func (a *APIActors) SetFilterAll() *APIActors {
	a.Filter = append(a.Filter, ActorsFilterAll)
	return a
}

func (a *APIActors) SetFilterPlayable() *APIActors {
	a.Filter = append(a.Filter, ActorsFilterPlayable)
	return a
}

func (a *APIActors) SetFilterSingle() *APIActors {
	a.Filter = append(a.Filter, ActorsFilterSingle)
	return a
}

func (a *APIActors) SetFilterCanDownload() *APIActors {
	a.Filter = append(a.Filter, ActorsFilterCanDownload)
	return a
}

func (a *APIActors) SetFilterHasZH() *APIActors {
	a.Filter = append(a.Filter, ActorsFilterHasZH)
	return a
}

func (a *APIActors) Get() ([]*JavDB, error) {
	return a.base.Get(a)
}
