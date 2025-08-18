package javdbapi

import (
	"testing"
)

func TestAPIMakers(t *testing.T) {
	client := newTestClient()
	_, err := client.GetMakers().
		SetDebug().
		SetMaker("7R"). // S1 NO.1 STYLE
		SetFilterCNSub().
		Get()
	if err != nil {
		panic(err)
	}
}
