package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"

	javdbapi "github.com/RPbro/javdbapi"
	"github.com/RPbro/javdbapi/internal/fetch"
)

func newCookieHTTPClient(filename, baseURL string, config javdbapi.HTTPConfig) (*http.Client, error) {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	origin, err := url.Parse(baseURL)
	if err != nil || origin.Scheme != "https" || origin.Hostname() == "" || origin.User != nil {
		return nil, errors.New("--cookie-file requires a valid HTTPS --base-url")
	}

	timeout := config.Timeout
	switch {
	case timeout < 0:
		return nil, fmt.Errorf("%w: negative HTTP timeout", javdbapi.ErrInvalidConfig)
	case timeout == 0:
		timeout = defaultHTTPTimeout
	}
	client, err := fetch.NewHTTPClient(timeout, config.ProxyURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", javdbapi.ErrInvalidConfig, err)
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		var pathErr *os.PathError
		if errors.As(err, &pathErr) {
			err = pathErr.Err
		}
		return nil, fmt.Errorf("read --cookie-file: %w", err)
	}
	cookies, err := http.ParseCookie(strings.TrimSpace(string(data)))
	if err != nil {
		return nil, errors.New("invalid --cookie-file: expected one Cookie header value (name=value; ...)")
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	for _, cookie := range cookies {
		cookie.Path = "/"
		cookie.Secure = true
	}
	jar.SetCookies(origin, cookies)
	client.Jar = jar
	return client, nil
}
