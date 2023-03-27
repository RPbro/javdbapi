package javdbapi

import (
	"fmt"
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient()

	result, err := client.GetFirst().WithReviews().SetRaw("https://javdb008.com/v/09qz3").First()
	if err != nil {
		panic(err)
	}

	fmt.Println(result)
}
