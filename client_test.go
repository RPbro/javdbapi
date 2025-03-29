package javdbapi

import (
	"fmt"
	"testing"
)

const testDomain = "https://javdb456.com"

func TestNewClient(t *testing.T) {
	client := NewClient(WithDomain(testDomain))

	filter := Filter{
		HasZH:         true,
		HasMagnets:    true,
		RegexpMagnets: "-UC.torrent|无码|破解",
	}

	result, err := client.GetMakers().
		WithDetails().
		WithRandom().
		SetMaker("7R").
		SetFilter(filter).
		Get()
	if err != nil {
		panic(err)
	}

	for _, i := range result {
		fmt.Println(i.Magnets)
	}
}
