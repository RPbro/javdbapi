package javdbapi

import (
	"testing"
)

func TestAPIRaw(t *testing.T) {
	client := newTestClient()
	_, err := client.GetRaw().
		SetDebug().
		SetRaw("https://javdb.com/makers/OXz?f=cnsub").
		Get()
	if err != nil {
		panic(err)
	}
}
