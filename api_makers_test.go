package javdbapi

import (
	"fmt"
	"testing"
)

func TestAPIMakers(t *testing.T) {
	client := newTestClient()
	result, err := client.GetMakers().
		SetMaker("7R").
		SetFilterCNSub().
		Get()
	if err != nil {
		panic(err)
	}
	for _, r := range result {
		fmt.Println(client.domain + r.Path)
		fmt.Printf("%+v\n", r)
	}
}
