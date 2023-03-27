package javdbapi

import (
	"fmt"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	domain := "https://javdb008.com"
	timeout := time.Second * 30

	client := NewClient(
		WithDomain(domain),
		WithTimeout(timeout),
	)

	filter := Filter{
		HasZH:      true,
		HasPreview: true,
		HasPics:    true,
		HasMagnets: true,
		HasReviews: true,
	}

	results, err := client.GetRaw().
		WithDetails().
		WithReviews().
		SetRaw("https://javdb.com/tags?c10=1,2,3,5").
		SetPage(1).
		SetLimit(10).
		SetFilter(filter).
		Get()
	if err != nil {
		panic(err)
	}

	for _, v := range results {
		fmt.Println(v)
	}
	fmt.Println(len(results))
}
