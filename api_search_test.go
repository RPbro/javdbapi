package javdbapi

import (
	"testing"
)

func TestAPISearch(t *testing.T) {
	client := newTestClient()
	_, err := client.GetSearch().
		SetDebug().
		SetQuery("VR").
		Get()
	if err != nil {
		panic(err)
	}
}
