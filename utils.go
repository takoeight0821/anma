package main

import (
	"fmt"

	"golang.org/x/exp/constraints"
	"golang.org/x/exp/slices"
)

func all[T any](slice []T, pred func(T) bool) bool {
	for _, v := range slice {
		if !pred(v) {
			return false
		}
	}
	return true
}

func orderedFor[I constraints.Ordered, V any](m map[I]V, f func(I, V)) {
	keys := make([]I, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		f(k, m[k])
	}
}

func errorAt(t Token, msg string) error {
	if t.Kind == EOF {
		return fmt.Errorf("at end: %s", msg)
	}
	return fmt.Errorf("at %d: `%s`, %s", t.Line, t.Lexeme, msg)
}
