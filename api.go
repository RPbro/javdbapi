package javdbapi

import (
	"log"
	"net/url"
	"strconv"
	"strings"
)

type API struct {
	client      *Client
	withDetails bool
	withReviews bool
	withRandom  bool
	withDebug   bool
	Page        int
	Limit       int
}

func (a *API) WithDetails() *API {
	a.withDetails = true
	return a
}

func (a *API) WithReviews() *API {
	a.withReviews = true
	return a
}

func (a *API) WithRandom() *API {
	a.withRandom = true
	return a
}

func (a *API) WithDebug() *API {
	a.withDebug = true
	return a
}

func (a *API) SetPage(page int) *API {
	if page > defaultPageMax {
		page = defaultPage
	}
	a.Page = page
	return a
}

func (a *API) SetLimit(limit int) *API {
	if limit > 0 {
		a.Limit = limit
	}
	return a
}

func (a *API) Get(t interface{}) ([]*JavDB, error) {
	u, err := url.Parse(a.client.Domain)
	if err != nil {
		return nil, err
	}

	switch p := t.(type) {
	case *APIRaw:
		u, err = url.Parse(p.Raw)
		if err != nil {
			return nil, err
		}
	case *APIHomes:
		u.Path = p.Category
		u = urlQueriesSet(u, map[string]string{
			"vst": p.SortBy,
			"vft": p.FilterBy,
		})
	case *APIRankings:
		u.Path = PathRankings
		u = urlQueriesSet(u, map[string]string{
			"t": p.Category,
			"p": p.Time,
		})
	case *APIMakers:
		u.Path = PathMakers + "/" + p.Maker
		u = urlQueriesSet(u, map[string]string{
			"f": p.Filter,
		})
	case *APIActors:
		u.Path = PathActors + "/" + p.Actor
		u = urlQueriesSet(u, map[string]string{
			"t": strings.Join(duplicateRemoving(p.Filter), ","),
		})
	default:
		return nil, err
	}

	a.Page = finalPage(a.Page, a.withRandom)
	u = urlQuerySet(u, "page", strconv.Itoa(a.Page))

	j := &JavDB{
		req: &request{
			client: a.client.HTTP,
			ua:     a.client.UserAgent,
			limit:  a.Limit,
			url:    u.String(),
		},
	}

	results, err := j.loadList()
	if err != nil {
		return nil, err
	}

	if a.withDetails {
		for _, i := range results {
			i.req.url = a.client.Domain + i.Path
			err = i.loadDetails()
			if err != nil {
				return nil, err
			}
		}
	}
	if a.withReviews {
		for _, i := range results {
			i.req.url = a.client.Domain + i.Path + PathReviews
			err = i.loadReviews()
			if err != nil {
				return nil, err
			}
		}
	}

	if a.withDebug {
		for _, i := range results {
			log.Printf("%+v\n", i)
		}
		log.Printf("%+v\n", a)
	}

	return results, nil
}
