package javdbapi

import (
	"crypto/tls"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type API struct {
	client *Client
	debug  bool
	random bool
	page   int
	limit  int
	filter *Filter
}

func (a *API) SetDebug() *API {
	a.debug = true
	return a
}

func (a *API) SetRandom() *API {
	a.random = true
	return a
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

	if a.random {
		r := rand.New(rand.NewSource(time.Now().Unix()))
		a.page = r.Intn(defaultPageMax)
	}
	if a.page != 0 {
		u = urlQuerySet(u, "page", strconv.Itoa(a.page))
	}
	u = urlQuerySet(u, "locale", "zh")
	u = urlQueryClean(u)

	hc, err := a.newHttpClient()
	if err != nil {
		return nil, err
	}

	m := map[string]*Item{}

	var listCount, detailCount, reviewsCount, resultFilterCount, finalCount int

	resp, err := a.fetchList(hc, u.String())
	if err != nil {
		return nil, err
	}
	for _, item := range resp {
		if _, ok := m[item.ID]; !ok {
			m[item.ID] = item
			listCount++
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
		detailCount++

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
		reviewsCount++
	}

	if a.filter != nil {
		if len(a.filter.ResultRegexpMagnets) > 0 {
			for _, item := range m {
				a.filter.pass = true
				a.filter.checkResultRegexpMagnets(item.Magnets)
				if a.filter.pass {
					resultFilterCount++
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
		finalCount++
	}

	for k, v := range result {
		a.log("item %d %s: %v", k+1, a.client.domain+v.Path, v)
	}
	a.log("list count: %d", listCount)
	a.log("detail count: %d", detailCount)
	a.log("reviews count: %d", reviewsCount)
	a.log("final count: %d, limit: %d", finalCount, a.limit)

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

	a.log("item %s: %v", link, item)

	return item, nil
}

func (a *API) newHttpClient() (*http.Client, error) {
	a.log("client: %+v", a.client)
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

func (a *API) request(hc *http.Client, link string) (*goquery.Document, error) {
	a.log("fetching: %s", link)
	resp, err := hc.Get(link)
	if err != nil {
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return doc, nil
}

func (a *API) log(format string, v ...any) {
	if !a.debug {
		return
	}
	log.Printf("[debug] "+format, v...)
}
