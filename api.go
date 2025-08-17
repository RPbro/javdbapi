package javdbapi

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type API struct {
	client *Client
	page   int
	limit  int
	filter *Filter
}

func (a *API) SetPage(page int) *API {
	if page > defaultPageMax || page < defaultPage {
		page = defaultPage
	}
	a.page = page
	return a
}

func (a *API) SetLimit(limit int) *API {
	if limit > 0 {
		a.limit = limit
	}
	return a
}

func (a *API) SetFilter(filter Filter) *API {
	a.filter = &Filter{
		ScoreGT:       filter.ScoreGT,
		ScoreLT:       filter.ScoreLT,
		ScoreCountGT:  filter.ScoreCountGT,
		ScoreCountLT:  filter.ScoreCountLT,
		PubDateBefore: filter.PubDateBefore,
		PubDateAfter:  filter.PubDateAfter,
		HasSubtitle:   filter.HasSubtitle,

		HasPreview:  filter.HasPreview,
		ActorsIn:    filter.ActorsIn,
		ActorsNotIn: filter.ActorsNotIn,
		TagsIn:      filter.TagsIn,
		TagsNotIn:   filter.TagsNotIn,
		HasPics:     filter.HasPics,
		HasMagnets:  filter.HasMagnets,

		HasReviews: filter.HasReviews,

		ResultRegexpMagnets: filter.ResultRegexpMagnets,
	}
	return a
}

func (a *API) Get(t any) ([]*Item, error) {
	var result []*Item

	u, err := url.Parse(a.client.domain)
	if err != nil {
		return nil, err
	}

	switch p := t.(type) {
	case *APIRaw:
		u, err = url.Parse(p.Raw)
		if err != nil {
			return nil, err
		}
	case *APIHome:
		u = u.JoinPath(PathHome, p.Type)
		u = urlQueriesSet(u, map[string]string{
			"vst": p.Sort,
			"vft": p.Filter,
		})
	case *APIRankings:
		u = u.JoinPath(PathRankings)
		u = urlQueriesSet(u, map[string]string{
			"t": p.Type,
			"p": p.Period,
		})
	case *APIMakers:
		u = u.JoinPath(PathMakers, p.Maker)
		u = urlQueriesSet(u, map[string]string{
			"f": p.Filter,
		})
	case *APIActors:
		u = u.JoinPath(PathActors, p.Actor)
		u = urlQueriesSet(u, map[string]string{
			"t": strings.Join(sliceDuplicateRemoving(p.Filter), ","),
		})
	case *APISearch:
		u = u.JoinPath(PathSearch)
		u = urlQueriesSet(u, map[string]string{
			"q": p.Query,
			"f": "all",
		})
	default:
		return nil, nil
	}

	if a.page != 0 {
		u = urlQuerySet(u, "page", strconv.Itoa(a.page))
	}
	u = urlQuerySet(u, "locale", "zh")

	hc, err := a.newHttpClient()
	if err != nil {
		return nil, err
	}

	m := map[string]*Item{}

	resp, err := a.fetchList(hc, u.String())
	for _, item := range resp {
		if _, ok := m[item.ID]; !ok {
			m[item.ID] = item
		}

		item, err = a.fetchDetail(hc, a.client.domain+item.Path, item)
		if err != nil {
			return nil, err
		}
		if item.remove {
			if _, ok := m[item.ID]; ok {
				delete(m, item.ID)
			}
			continue
		}

		item, err = a.fetchReviews(hc, a.client.domain+item.Path+PathReviews, item)
		if err != nil {
			return nil, err
		}
		if item.remove {
			if _, ok := m[item.ID]; ok {
				delete(m, item.ID)
			}
			continue
		}
	}

	if a.filter != nil {
		if len(a.filter.ResultRegexpMagnets) > 0 {
			for _, item := range m {
				a.filter.pass = true
				a.filter.checkResultRegexpMagnets(item.Magnets)
				if a.filter.pass {
					continue
				}
				if _, ok := m[item.ID]; ok {
					delete(m, item.ID)
				}
			}
		}
	}

	for _, item := range m {
		if a.limit > 0 && len(result) >= a.limit {
			continue
		}
		result = append(result, item)
	}

	return result, nil
}

func (a *API) First(link string) (*Item, error) {
	hc, err := a.newHttpClient()
	if err != nil {
		return nil, err
	}

	item, err := a.fetchDetail(hc, link, nil)
	if err != nil {
		return nil, err
	}

	item, err = a.fetchReviews(hc, a.client.domain+item.Path+PathReviews, item)
	if err != nil {
		return nil, err
	}

	return item, nil
}

func (a *API) newHttpClient() (*http.Client, error) {
	hc := &http.Client{
		Timeout: a.client.timeout,
	}
	if len(a.client.proxy) > 0 {
		proxyURL, err := url.Parse(a.client.proxy)
		if err != nil {
			return nil, err
		}
		tr := &http.Transport{
			Proxy:           http.ProxyURL(proxyURL),
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		hc.Transport = tr
	}
	return hc, nil
}
