package javdbapi

import (
	"fmt"
	"testing"
)

func TestAPISearch(t *testing.T) {
	client := newTestClient()
	result, err := client.GetSearch().
		SetQuery("VR").
		Get()
	if err != nil {
		panic(err)
	}
	for _, r := range result {
		fmt.Println(client.domain + r.Path)
		fmt.Printf("%+v\n", r)
	}
}
