package utils

import (
	"fmt"
	"os"

	"github.com/takoeight0821/anma/token"
	"golang.org/x/exp/constraints"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

func All[T any](slice []T, pred func(T) bool) bool {
	for _, v := range slice {
		if !pred(v) {
			return false
		}
	}
	return true
}

func OrderedFor[I constraints.Ordered, V any](m map[I]V, f func(I, V)) {
	keys := make([]I, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		f(k, m[k])
	}
}

// MsgAt returns a string that describes the location of the token.
func MsgAt(t token.Token, msg string) string {
	if t.Kind == token.EOF {
		return fmt.Sprintf("at end: %s", msg)
	}
	return fmt.Sprintf("at %d: `%s`, %s", t.Line, t.Lexeme, msg)
}

type TestData struct {
	Label    string
	Enable   bool
	Input    string
	Expected map[string]string
}

func ReadTestData() []TestData {
	s, err := os.ReadFile("../testdata/testcase.yaml")
	if err != nil {
		panic(err)
	}

	var data []TestData
	if err := yaml.Unmarshal(s, &data); err != nil {
		panic(err)
	}

	// Remove disabled test cases.
	i := 0
	for _, d := range data {
		if d.Enable {
			data[i] = d
			i++
		}
	}
	data = data[:i]

	return data
}
