package javdbapi

import (
	"net/url"
	"strings"
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

func urlQueryClean(u *url.URL) *url.URL {
	q := u.Query()
	for key, values := range q {
		allEmpty := true
		for _, value := range values {
			if strings.TrimSpace(value) != "" {
				allEmpty = false
				break
			}
		}
		if allEmpty {
			delete(q, key)
		}
	}
	u.RawQuery = q.Encode()
	return u
}
