package javdbapi

import (
	"testing"
)

func TestAPIHome(t *testing.T) {
	client := newTestClient()
	_, err := client.GetHome().
		SetDebug().
		SetTypeCensored().
		SetFilterCNSub().
		Get()
	if err != nil {
		panic(err)
	}
}
