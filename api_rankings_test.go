package javdbapi

import (
	"testing"
)

func TestAPIRankings(t *testing.T) {
	client := newTestClient()
	_, err := client.GetRankings().
		SetDebug().
		SetPeriodDaily().
		SetTypeCensored().
		Get()
	if err != nil {
		panic(err)
	}
}
