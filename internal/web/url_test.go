package web

import (
	"net/url"
	"strings"
	"testing"
)

func TestBuildURLSkipsEmptyValues(t *testing.T) {
	base, err := url.Parse("https://example.com")
	if err != nil {
		t.Fatalf("parse base url: %v", err)
	}

	got, err := BuildURL(base, "/search", map[string]string{
		"q":     "abc",
		"empty": "",
		"x":     "1",
	})
	if err != nil {
		t.Fatalf("BuildURL: %v", err)
	}

	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse result: %v", err)
	}
	if parsed.Path != "/search" {
		t.Fatalf("path: got %q want %q", parsed.Path, "/search")
	}

	q := parsed.Query()
	if q.Get("q") != "abc" {
		t.Fatalf("q: got %q want %q", q.Get("q"), "abc")
	}
	if q.Get("x") != "1" {
		t.Fatalf("x: got %q want %q", q.Get("x"), "1")
	}
	if _, ok := q["empty"]; ok {
		t.Fatalf("empty query param should be skipped")
	}
}

func TestBuildURLRejectsInvalidBaseURL(t *testing.T) {
	cases := []struct {
		name string
		base *url.URL
	}{
		{name: "nil", base: nil},
		{name: "relative", base: &url.URL{Path: "/relative"}},
		{name: "missing_scheme", base: &url.URL{Host: "example.com", Path: "/"}},
		{name: "missing_host", base: &url.URL{Scheme: "https", Path: "/"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := BuildURL(tc.base, "/x", nil)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
		})
	}
}

func TestBuildURLHandlesRawPathQueryAndFragment(t *testing.T) {
	base, err := url.Parse("https://example.com")
	if err != nil {
		t.Fatalf("parse base url: %v", err)
	}

	got, err := BuildURL(base, "/search?q=abc#top", map[string]string{
		"x": "1",
	})
	if err != nil {
		t.Fatalf("BuildURL: %v", err)
	}

	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse result: %v", err)
	}
	if parsed.Path != "/search" {
		t.Fatalf("path: got %q want %q", parsed.Path, "/search")
	}
	if parsed.Fragment != "top" {
		t.Fatalf("fragment: got %q want %q", parsed.Fragment, "top")
	}

	q := parsed.Query()
	if q.Get("q") != "abc" {
		t.Fatalf("q: got %q want %q", q.Get("q"), "abc")
	}
	if q.Get("x") != "1" {
		t.Fatalf("x: got %q want %q", q.Get("x"), "1")
	}
}

func TestBuildURLPreservesEscapedPath(t *testing.T) {
	base, err := url.Parse("https://example.com")
	if err != nil {
		t.Fatalf("parse base url: %v", err)
	}

	got, err := BuildURL(base, "/a%2Fb", nil)
	if err != nil {
		t.Fatalf("BuildURL: %v", err)
	}
	if strings.Contains(got, "%252F") {
		t.Fatalf("should not double-escape path: %q", got)
	}

	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse result: %v", err)
	}
	if parsed.EscapedPath() != "/a%2Fb" {
		t.Fatalf("escaped path: got %q want %q (full=%q)", parsed.EscapedPath(), "/a%2Fb", got)
	}
}
