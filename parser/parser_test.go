package parser_test

import (
	"strings"
	"testing"

	"github.com/takoeight0821/anma/driver"
	"github.com/takoeight0821/anma/utils"
)

func TestParseFromTestData(t *testing.T) {
	testcases := utils.ReadTestData()
	for _, testcase := range testcases {
		if expected, ok := testcase.Expected["parser"]; ok {
			newCompleteParse(t, testcase.Label, testcase.Input, expected)
		} else {
			newCompleteParse(t, testcase.Label, testcase.Input, "no expected value")
		}
	}
}

func newCompleteParse(t *testing.T, label, input, expected string) {
	r := driver.NewPassRunner()

	nodes, err := r.RunSource(input)
	if err != nil {
		t.Errorf("RunSource %s returned error: %v", label, err)
	}

	var b strings.Builder
	for _, node := range nodes {
		b.WriteString(node.String())
		b.WriteString("\n")
	}

	actual := b.String()
	if actual != expected {
		t.Errorf("RunSource %s returned:\n%s\n\nexpected:\n%s", label, actual, expected)
	}
}
