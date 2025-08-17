package javdbapi

import (
	"fmt"
	"testing"
)

func TestAPIRaw(t *testing.T) {
	client := newTestClient()
	result, err := client.GetRaw().
		SetRaw("https://javdb.com/makers/OXz?f=cnsub").
		Get()
	if err != nil {
		panic(err)
	}
	for _, r := range result {
		fmt.Println(client.domain + r.Path)
		fmt.Printf("%+v\n", r)
	}
}
