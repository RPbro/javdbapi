package javdbapi

import (
	"testing"
)

func TestAPIFirst(t *testing.T) {
	client := newTestClient()
	_, err := client.GetFirst().
		SetDebug().
		SetRaw("https://javdb.com/v/ZNdEbV").
		First()
	if err != nil {
		panic(err)
	}
}
