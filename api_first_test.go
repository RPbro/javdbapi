package javdbapi

import (
	"fmt"
	"testing"
)

func TestAPIFirst(t *testing.T) {
	client := newTestClient()
	result, err := client.GetFirst().
		SetRaw("https://javdb.com/v/ZNdEbV").
		First()
	if err != nil {
		panic(err)
	}
	fmt.Println(client.domain + result.Path)
	fmt.Printf("%+v\n", result)
}
