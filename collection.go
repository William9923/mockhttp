package mockhttp

import "fmt"

// Collection Utility Function: in
// Check if current object exist in a collection, similar to how Python "in" works.
//
// Accept any comparable object
// ex:
// collection: ["a", "b", "c"]
// "d" in collection => false
// "a" in collection => true
func in[T comparable](current T, collections []T) bool {
	return some[T](collections, func(param T) bool {
		return param == current
	})
}

// Collection Utility Function: some
// Check if any of the object in current collection
// satisfy the objective (defined by a custom function that return true/false)
// true means satisfy the objective
// false means not satisfy the objective
//
// Accept any object
// ex:
// collection: [1, 2, 3]
// custom function: greater than 4 (func (num int) bool { return num > 4 })
// output: false (no digit greater than 4 in collection)
//
// collection: [1, 2, 3]
// custom function: equal 3 (func (num int) bool { return num == 3 })
// output: true (there are digit with value equal to 3)
func some[T any](collections []T, fn func(T) bool) bool {
	for _, data := range collections {
		if fn(data) {
			return true
		}
	}
	return false
}

// Collection Utility Function: all
// Check if all of the object in current collection
// satisfy the objective (defined by a custom function that return true/false)
// true means satisfy the objective
// false means not satisfy the objective
//
// Accept any object
// ex:
// collection: [1, 2, 3]
// custom function: less than 4 (func (num int) bool { return num < 4 })
// output: true (no digit greater than 4 in collection)
//
// collection: [1, 2, 3]
// custom function: equal 3 (func (num int) bool { return num == 3 })
// output: false (there are only 1 digit with value equal to 3)
func all[T any](collection []T, fn func(T) bool) bool {
	for _, data := range collection {
		if !fn(data) {
			return false
		}
	}
	return true
}

// Collection Utility Function: findFirst
// Find the first object in current collection that
// satisfy the objective (defined by a custom function that return true/false)
// true means satisfy the objective
// false means not satisfy the objective
//
// Accept any object
// ex:
// collection: [1, 2, 3]
// custom function: less than 4 (func (num int) bool { return num < 4 })
// output: 1 (keep the ordering based on the collection ordering)
//
// collection: [1, 2, 3]
// custom function: equal 4 (func (num int) bool { return num == 4 })
// output: error (no match found)
func findFirst[T any](collections []T, fn func(T) bool) (T, error) {
	for _, data := range collections {
		if fn(data) {
			return data, nil
		}
	}
	var empty T
	return empty, fmt.Errorf("no match found")
}

// Collection Utility Function: merge
// Merge / flatten more than 1 collections with same type into 1 collection
//
// Accept any object
// ex:
// collections: [1, 2, 3], [4, 5, 6]
// output: [1, 2, 3, 4, 5, 6]
func merge[T any](collections ...[]T) []T {
	var merged []T
	for _, collection := range collections {
		merged = append(merged, collection...)
	}
	return merged
}

// Collection Utility Function: filter
// Return elements that satisfy defined conditions
// from a collections
//
// Accept any object
// ex:
// collections: [1, 2, 3]
// custom function: less than 3 (func (num int) bool { return num < 3 })
// output: [1, 2]
func filter[T any](collections []T, condition func(T) bool) []T {
	var result []T
	for _, data := range collections {
		if condition(data) {
			result = append(result, data)
		}
	}
	return result
}
