package javdbapi

import (
	"regexp"
	"strconv"
	"strings"
)

func strTrimSpace(s string) string {
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, " ", "")
	return s
}

func strIsInt(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

func StrIsMagnet(s string) bool {
	re := regexp.MustCompile(`^magnet:\?xt=urn:btih:[0-9a-fA-F]{32,40}.*$`)
	return re.MatchString(s)
}

type MSet map[any]struct{}

func duplicateRemoving[T any](s []T) []T {
	res := make([]T, 0, len(s))
	m := make(MSet)
	for _, t := range s {
		if _, ok := m[t]; !ok {
			res = append(res, t)
			m[t] = struct{}{}
		}
	}

	return res
}
