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

func completeResolve(t *testing.T, input, expected string) {
	runner := driver.NewPassRunner()
	runner.AddPass(codata.Flat{})
	runner.AddPass(infix.NewInfixResolver())
	runner.AddPass(nameresolve.NewResolver())

	nodes, err := runner.RunSource(input)
	if err != nil {
		t.Errorf("RunSource returned error: %v", err)
	}

	var b strings.Builder
	for _, node := range nodes {
		b.WriteString(node.String())
		b.WriteString("\n")
	}
	actual := b.String()

	if actual != expected {
		t.Errorf("RunSource returned:\n%s\n\nexpected:\n%s", actual, expected)
	}
}

func TestResolve(t *testing.T) {
	testcases := utils.ReadTestData()
	for _, testcase := range testcases {
		if expected, ok := testcase.Expected["nameresolve"]; ok {
			completeResolve(t, testcase.Input, expected)
		} else {
			completeResolve(t, testcase.Input, "no expected value")
		}
	}
}
