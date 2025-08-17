package javdbapi

import (
	"fmt"
	"testing"
)

func TestAPIActors(t *testing.T) {
	client := newTestClient()
	result, err := client.GetActors().
		SetActor("neRNX"). // 瀬戸環奈
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
