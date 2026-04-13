package web

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunnerGetReturnsBody(t *testing.T) {
	t.Parallel()

	const wantBody = "hello"
	const wantUA = "test-agent"

	uaCh := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uaCh <- r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(wantBody))
	}))
	t.Cleanup(server.Close)

	runner, err := NewRunner(2*time.Second, "", wantUA, false)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	got, err := runner.Get(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got) != wantBody {
		t.Fatalf("body: got %q want %q", string(got), wantBody)
	}

	select {
	case gotUA := <-uaCh:
		if gotUA != wantUA {
			t.Fatalf("User-Agent: got %q want %q", gotUA, wantUA)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for server to receive request")
	}
}

func TestRunnerGetRejectsUnexpectedStatus(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	}))
	t.Cleanup(server.Close)

	runner, err := NewRunner(2*time.Second, "", "test-agent", false)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	_, err = runner.Get(context.Background(), server.URL)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestRunnerGetUnexpectedStatusIsTyped(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	}))
	t.Cleanup(server.Close)

	runner, err := NewRunner(2*time.Second, "", "test-agent", false)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	_, err = runner.Get(context.Background(), server.URL)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	var use *UnexpectedStatusError
	if !errors.As(err, &use) {
		t.Fatalf("expected UnexpectedStatusError, got %T: %v", err, err)
	}
	if use.StatusCode != http.StatusNotFound {
		t.Fatalf("status code: got %d want %d", use.StatusCode, http.StatusNotFound)
	}
	if use.URL != server.URL {
		t.Fatalf("url: got %q want %q", use.URL, server.URL)
	}
}

func TestNewRunnerRejectsInvalidProxyURL(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		proxyURL string
		wantErr  bool
	}{
		{name: "empty_ok", proxyURL: "", wantErr: false},
		{name: "localhost_rejected", proxyURL: "localhost", wantErr: true},
		{name: "http_missing_host_rejected", proxyURL: "http://", wantErr: true},
		{name: "http_empty_hostname_rejected", proxyURL: "http://:8080", wantErr: true},
		{name: "ftp_rejected", proxyURL: "ftp://127.0.0.1:21", wantErr: true},
		{name: "socks4_rejected", proxyURL: "socks4://127.0.0.1:1080", wantErr: true},
		{name: "socks5_ok", proxyURL: "socks5://127.0.0.1:1080", wantErr: false},
		{name: "socks5h_ok", proxyURL: "socks5h://127.0.0.1:1080", wantErr: false},
		{name: "valid_http_ok", proxyURL: "http://127.0.0.1:8080", wantErr: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r, err := NewRunner(2*time.Second, tc.proxyURL, "ua", false)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil runner=%v", r)
				}
				return
			}

			if err != nil {
				t.Fatalf("NewRunner: %v", err)
			}
			if r == nil {
				t.Fatalf("expected runner, got nil")
			}
		})
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

type trackingBody struct {
	data   []byte
	offset int

	readCalls int64
	closed    int64
}

func (b *trackingBody) Read(p []byte) (int, error) {
	atomic.AddInt64(&b.readCalls, 1)
	if b.offset >= len(b.data) {
		return 0, io.EOF
	}
	n := copy(p, b.data[b.offset:])
	b.offset += n
	return n, nil
}

func (b *trackingBody) Close() error {
	atomic.StoreInt64(&b.closed, 1)
	return nil
}

func TestRunnerGetDrainsBodyOnNon2xx(t *testing.T) {
	t.Parallel()

	body := &trackingBody{data: []byte("not found")}
	rt := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       body,
			Header:     make(http.Header),
			Request:    r,
		}, nil
	})

	runner := &Runner{
		client:    &http.Client{Transport: rt},
		userAgent: "ua",
		debug:     false,
	}

	_, err := runner.Get(context.Background(), "http://example.com")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if atomic.LoadInt64(&body.readCalls) == 0 {
		t.Fatalf("expected response body to be drained on non-2xx status")
	}
	if atomic.LoadInt64(&body.closed) == 0 {
		t.Fatalf("expected response body to be closed")
	}
}
