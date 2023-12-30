package mockhttp

import "fmt"

func in[T comparable](current T, collections []T) bool {
	return satisfyAtLeastOne[T](collections, func(param T) bool {
		return param == current
	})
}

func satisfyAtLeastOne[T any](collections []T, fn func(T) bool) bool {
	for _, data := range collections {
		if fn(data) {
			return true
		}
	}
	return false
}

func satisfyEvery[T any](collection []T, fn func(T) bool) bool {
	for _, data := range collection {
		if !fn(data) {
			return false
		}
	}
	return true
}

func some[T any](collections []T, fn func(T) bool) bool {
	for _, data := range collections {
		if fn(data) {
			return true
		}
	}
	return false
}

func every[T any](collection []T, fn func(T) bool) bool {
	for _, data := range collection {
		if !fn(data) {
			return false
		}
	}
	return true
}

func find[T any](collections []T, fn func(T) bool) (T, error) {
	for _, data := range collections {
		if fn(data) {
			return data, nil
		}
	}
	var empty T
	return empty, fmt.Errorf("no match found")
}
