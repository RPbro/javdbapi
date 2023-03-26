package javdbapi

import (
	"fmt"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	domain := "https://javdb008.com"
	timeout := time.Second * 30

	client := NewClient(
		WithDomain(domain),
		WithTimeout(timeout),
	)
	result, err := client.GetRankings().
		WithDebug().
		SetCategoryCensored().
		SetTimeMonthly().
		SetPage(1).
		SetLimit(1).
		Get()
	if err != nil {
		panic(err)
	}

	fmt.Println(len(result))
}
