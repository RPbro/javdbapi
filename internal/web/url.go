package web

import (
	"fmt"
	"net/url"
	"strings"
)

func BuildURL(base *url.URL, rawPath string, params map[string]string) (string, error) {
	if base == nil || base.Scheme == "" || base.Host == "" {
		return "", fmt.Errorf("invalid base url")
	}

	u := *base

	q := u.Query()

	if rawPath != "" {
		parsed, err := url.Parse(rawPath)
		if err != nil {
			return "", fmt.Errorf("parse raw path: %w", err)
		}

		if parsed.Path != "" || parsed.RawPath != "" {
			joinedEscaped := joinPath(u.EscapedPath(), parsed.EscapedPath())
			unescaped, err := url.PathUnescape(joinedEscaped)
			if err != nil {
				return "", fmt.Errorf("unescape joined path: %w", err)
			}
			u.Path = unescaped
			u.RawPath = joinedEscaped
		}
		if parsed.Fragment != "" {
			u.Fragment = parsed.Fragment
		}

		for k, values := range parsed.Query() {
			for _, v := range values {
				if v == "" {
					continue
				}
				q.Add(k, v)
			}
		}
	}

	for k, v := range params {
		if v == "" {
			continue
		}
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	return u.String(), nil
}

func joinPath(basePath string, rawPath string) string {
	bp := basePath
	if bp == "" {
		bp = "/"
	}

	rp := rawPath
	if !strings.HasPrefix(rp, "/") {
		rp = "/" + rp
	}

	if bp == "/" {
		return rp
	}

	joined := strings.TrimSuffix(bp, "/") + "/" + strings.TrimPrefix(rp, "/")
	if !strings.HasPrefix(joined, "/") {
		joined = "/" + joined
	}
	return joined
}
