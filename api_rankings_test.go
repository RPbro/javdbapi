package javdbapi

import (
	"fmt"
	"testing"
)

func TestAPIRankings(t *testing.T) {
	client := newTestClient()
	result, err := client.GetRankings().
		SetPeriodDaily().
		SetTypeCensored().
		Get()
	if err != nil {
		panic(err)
	}
	for _, r := range result {
		fmt.Println(client.domain + r.Path)
		fmt.Printf("%+v\n", r)
	}
}
