package infix_test

import (
	"strings"
	"testing"

	"github.com/takoeight0821/anma/codata"
	"github.com/takoeight0821/anma/driver"
	"github.com/takoeight0821/anma/infix"
	"github.com/takoeight0821/anma/utils"
)

func TestInfix(t *testing.T) {
	testcases := utils.ReadTestData()

	for _, testcase := range testcases {
		if expected, ok := testcase.Expected["infix"]; ok {
			completeInfix(t, testcase.Label, testcase.Input, expected)
		} else {
			completeInfix(t, testcase.Label, testcase.Input, "no expected value")
		}
	}
}

func completeInfix(t *testing.T, label, input, expected string) {
	runner := driver.NewPassRunner()
	runner.AddPass(codata.Flat{})
	runner.AddPass(infix.NewInfixResolver())

	nodes, err := runner.RunSource(input)
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
		t.Errorf("RunSource %s expected -> actual\n%s", label, utils.SprintDiff(expected, actual))
	}
}
