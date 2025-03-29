package javdbapi

import (
	"reflect"
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

func strIsMatch(s string, reg string) bool {
	re := regexp.MustCompile(reg)
	return re.MatchString(s)
}

func strIsMagnet(s string) bool {
	reg := `^magnet:\?xt=urn:btih:[0-9a-fA-F]{32,40}.*$`
	return strIsMatch(s, reg)
}

type mSet map[any]struct{}

func sliceDuplicateRemoving[T any](s []T) []T {
	r := make([]T, 0, len(s))
	m := make(mSet)
	for _, t := range s {
		if _, ok := m[t]; !ok {
			r = append(r, t)
			m[t] = struct{}{}
		}
	}
	return r
}

func sliceRemoveIndex[T any](s []T, i int) []T {
	r := make([]T, 0)
	r = append(r, s[:i]...)
	return append(r, s[i+1:]...)
}

func sliceEqual[T comparable](s1 []T, s2 []T) bool {
	return reflect.DeepEqual(s1, s2)
}

func sliceElementInSlice[T comparable](e T, s []T) bool {
	for _, i := range s {
		if i == e {
			return true
		}
	}

	return false
}

func sliceContainsAny[T comparable](s1 []T, s2 []T) bool {
	pass := false
	for _, n := range s1 {
		if pass {
			continue
		}
		if sliceElementInSlice(n, s2) {
			pass = true
		}
	}

	return pass
}

func sliceIsSubset[T comparable](subset []T, superset []T) bool {
	if reflect.DeepEqual(subset, superset) {
		return true
	}
	check := make(map[T]bool)
	for _, v := range subset {
		check[v] = true
	}
	for _, v := range superset {
		if check[v] {
			delete(check, v)
		}
	}
	return len(check) == 0
}
