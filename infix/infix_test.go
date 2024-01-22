package infix_test

import (
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/takoeight0821/anma/codata"
	"github.com/takoeight0821/anma/driver"
	"github.com/takoeight0821/anma/infix"
	"github.com/takoeight0821/anma/utils"
)

func TestInfix(t *testing.T) {
	t.Parallel()
	s, err := os.ReadFile("../testdata/testcase.yaml")
	if err != nil {
		panic(err)
	}
	testcases := utils.ReadTestData(s)

	for _, testcase := range testcases {
		if expected, ok := testcase.Expected["infix"]; ok {
			completeInfix(t, testcase.Label, testcase.Input, expected)
		} else {
			completeInfix(t, testcase.Label, testcase.Input, "no expected value")
		}
	}
}

func BenchmarkInfix(b *testing.B) {
	s, err := os.ReadFile("../testdata/testcase.yaml")
	if err != nil {
		panic(err)
	}
	testcases := utils.ReadTestData(s)

	for _, testcase := range testcases {
		b.Run(testcase.Label, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				completeInfix(b, testcase.Label, testcase.Input, testcase.Expected["infix"])
			}
		})
	}
}

type reporter interface {
	Errorf(format string, args ...interface{})
}

func completeInfix(test reporter, label, input, expected string) {
	runner := driver.NewPassRunner()
	runner.AddPass(codata.Flat{})
	runner.AddPass(infix.NewInfixResolver())

	nodes, err := runner.RunSource(input)
	if err != nil {
		test.Errorf("Infix %s returned error: %v", label, err)
		return
	}

	if _, ok := test.(*testing.B); ok {
		// do nothing for benchmark
		return
	}

	var b strings.Builder
	for _, node := range nodes {
		b.WriteString(node.String())
		b.WriteString("\n")
	}
	actual := b.String()

	if diff := cmp.Diff(expected, actual); diff != "" {
		test.Errorf("Infix %s mismatch (-want +got):\n%s", label, diff)
	}
}
