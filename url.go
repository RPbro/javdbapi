package javdbapi

import (
	"net/url"
)

func urlQueriesSet(u *url.URL, queries map[string]string) *url.URL {
	q := u.Query()
	for k, v := range queries {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	return u
}

func urlQuerySet(u *url.URL, key string, value string) *url.URL {
	q := u.Query()
	q.Set(key, value)
	u.RawQuery = q.Encode()

	return u
}
