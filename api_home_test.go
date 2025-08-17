package javdbapi

import (
	"fmt"
	"testing"
)

func TestAPIHome(t *testing.T) {
	client := newTestClient()
	result, err := client.GetHome().
		SetTypeCensored().
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
