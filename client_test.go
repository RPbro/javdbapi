package javdbapi

import (
	"time"
)

func newTestClient() *Client {
	client := NewClient(
		WithDomain("https://javdb.com"),
		WithUserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"),
		WithProxy("http://10.10.10.20:1080"),
		WithTimeout(time.Second*30),
	)
	return client
}
