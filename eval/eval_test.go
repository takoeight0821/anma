package eval_test

import (
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/takoeight0821/anma/codata"
	"github.com/takoeight0821/anma/driver"
	"github.com/takoeight0821/anma/eval"
	"github.com/takoeight0821/anma/infix"
	"github.com/takoeight0821/anma/nameresolve"
	"github.com/takoeight0821/anma/token"
	"github.com/takoeight0821/anma/utils"
)

func TestEvalFromTestData(t *testing.T) {
	t.Parallel()
	s, err := os.ReadFile("../testdata/testcase.yaml")
	if err != nil {
		panic(err)
	}
	testcases := utils.ReadTestData(s)

	for _, testcase := range testcases {
		if expected, ok := testcase.Expected["eval"]; ok {
			completeEval(t, testcase.Label, testcase.Input, expected)
		} else {
			completeEval(t, testcase.Label, testcase.Input, "no expected value")
		}
	}
}

func BenchmarkFromTestData(b *testing.B) {
	s, err := os.ReadFile("../testdata/testcase.yaml")
	if err != nil {
		panic(err)
	}
	testcases := utils.ReadTestData(s)

	for _, testcase := range testcases {
		b.Run(testcase.Label, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				completeEval(b, testcase.Label, testcase.Input, testcase.Expected["eval"])
			}
		})
	}
}

type reporter interface {
	Errorf(format string, args ...interface{})
}

func completeEval(test reporter, label string, input string, expected string) {
	runner := driver.NewPassRunner()
	runner.AddPass(codata.Flat{})
	runner.AddPass(infix.NewInfixResolver())
	runner.AddPass(nameresolve.NewResolver())

	nodes, err := runner.RunSource(input)
	if err != nil {
		test.Errorf("Eval %s returned error: %v", label, err)

		return
	}

	evaluator := eval.NewEvaluator()
	values := make([]eval.Value, len(nodes))

	for i, node := range nodes {
		values[i], err = evaluator.Eval(node)
		if err != nil {
			test.Errorf("%s returned error: %v", label, err)

			return
		}
	}

	if main, ok := evaluator.SearchMain(); ok {
		top := token.Token{Kind: token.IDENT, Lexeme: "toplevel", Line: 0, Literal: -1}
		ret, err := main.Apply(top)
		if err != nil {
			test.Errorf("%s returned error: %v", label, err)

			return
		}

		if _, ok := test.(*testing.B); ok {
			// do nothing for benchmark
			return
		}

		actual := ret.String()

		if diff := cmp.Diff(expected, actual); diff != "" {
			test.Errorf("%s mismatch (-want +got):\n%s", label, diff)
		}
	} else {
		test.Errorf("%s does not have a main function", label)
	}
}
