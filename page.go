package javdbapi

import (
	"math/rand"
	"time"
)

func finalPage(page int, random bool) int {
	p := defaultPage
	switch {
	case page > 0:
		p = page
	case random:
		rand.Seed(time.Now().Unix())
		n := rand.Intn(defaultPageMax)
		p = n
	}

	return p
}
