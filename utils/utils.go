package utils

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/takoeight0821/anma/ast"
	"github.com/takoeight0821/anma/token"
	"gopkg.in/yaml.v3"
)

type PosError struct {
	Where token.Token
	Err   error
}

func (e PosError) Error() string {
	if e.Where.Kind == token.EOF {
		return fmt.Sprintf("at end: %s", e.Err.Error())
	}

	return fmt.Sprintf("at %d: `%s`, %s", e.Where.Line, e.Where.Lexeme, e.Err.Error())
}

type TestData struct {
	Label    string
	Enable   bool
	Input    string
	Expected map[string]string
}

func ReadTestData(s []byte) []TestData {
	var data []TestData
	if err := yaml.Unmarshal(s, &data); err != nil {
		panic(err)
	}

	// Remove disabled test cases.
	index := 0
	for _, d := range data {
		if d.Enable {
			data[index] = d
			index++
		}
	}
	data = data[:index]

	return data
}

type reporter interface {
	Logf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

type runner interface {
	RunSource(source string) ([]ast.Node, error)
}

func RunTest(runner runner, test reporter, label, input, expected string) {
	nodes, err := runner.RunSource(input)
	if err != nil {
		test.Errorf("%s returned error: %v", label, err)

		return
	}

	if _, ok := test.(*testing.B); ok {
		// do nothing for benchmark
		return
	}

	var builder strings.Builder
	for _, node := range nodes {
		builder.WriteString(node.String())
		builder.WriteString("\n")
	}
	actual := builder.String()

	if diff := cmp.Diff(expected, actual); diff != "" {
		test.Errorf("%s mismatch (-want +got):\n%s", label, diff)
		test.Logf("actual:\n%s", actual)
	}
}
