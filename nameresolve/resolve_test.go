package nameresolve_test

import (
	"strings"
	"testing"

	"github.com/takoeight0821/anma/codata"
	"github.com/takoeight0821/anma/driver"
	"github.com/takoeight0821/anma/infix"
	"github.com/takoeight0821/anma/nameresolve"
	"github.com/takoeight0821/anma/utils"
)

func TestResolve(t *testing.T) {
	testcases := utils.ReadTestData()
	for _, testcase := range testcases {
		if expected, ok := testcase.Expected["nameresolve"]; ok {
			completeResolve(t, testcase.Label, testcase.Input, expected)
		} else {
			completeResolve(t, testcase.Label, testcase.Input, "no expected value")
		}
	}
}

func completeResolve(t *testing.T, label, input, expected string) {
	runner := driver.NewPassRunner()
	runner.AddPass(codata.Flat{})
	runner.AddPass(infix.NewInfixResolver())
	runner.AddPass(nameresolve.NewResolver())

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