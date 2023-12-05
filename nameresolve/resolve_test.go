package nameresolve_test

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/takoeight0821/anma/codata"
	"github.com/takoeight0821/anma/driver"
	"github.com/takoeight0821/anma/infix"
	"github.com/takoeight0821/anma/nameresolve"
	"github.com/takoeight0821/anma/utils"
)

func TestResolve(t *testing.T) {
	t.Parallel()
	testcases := utils.ReadTestData()
	for _, testcase := range testcases {
		if expected, ok := testcase.Expected["nameresolve"]; ok {
			completeResolve(t, testcase.Label, testcase.Input, expected)
		} else {
			completeResolve(t, testcase.Label, testcase.Input, "no expected value")
		}
	}
}

func BenchmarkFromTestData(b *testing.B) {
	testcases := utils.ReadTestData()

	for _, testcase := range testcases {
		b.Run(testcase.Label, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				completeResolve(b, testcase.Label, testcase.Input, testcase.Expected["nameresolve"])
			}
		})
	}
}

type reporter interface {
	Errorf(format string, args ...interface{})
}

func completeResolve(t reporter, label, input, expected string) {
	runner := driver.NewPassRunner()
	runner.AddPass(codata.Flat{})
	runner.AddPass(infix.NewInfixResolver())
	runner.AddPass(nameresolve.NewResolver())

	nodes, err := runner.RunSource(input)
	if err != nil {
		t.Errorf("Resolve %s returned error: %v", label, err)
		return
	}

	var b strings.Builder
	for _, node := range nodes {
		b.WriteString(node.String())
		b.WriteString("\n")
	}
	actual := b.String()

	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Errorf("Resolve %s mismatch (-want +got):\n%s", label, diff)
	}
}
