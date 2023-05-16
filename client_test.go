package javdbapi

import (
	"fmt"
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient()

	result, err := client.GetSearch().
		SetQuery("PRED-483").
		SetLimit(3).
		Get()
	if err != nil {
		panic(err)
	}

	fmt.Println(result)
}
