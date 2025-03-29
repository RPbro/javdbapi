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
		r := rand.New(rand.NewSource(time.Now().Unix()))
		n := r.Intn(defaultPageMax)
		p = n
	}

	return p
}
