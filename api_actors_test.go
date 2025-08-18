package javdbapi

import (
	"testing"
)

func TestAPIActors(t *testing.T) {
	client := newTestClient()
	_, err := client.GetActors().
		SetDebug().
		SetActor("neRNX"). // 瀬戸環奈
		Get()
	if err != nil {
		panic(err)
	}
}
