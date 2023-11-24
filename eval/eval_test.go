package eval_test

import (
	"testing"

	"github.com/takoeight0821/anma/codata"
	"github.com/takoeight0821/anma/driver"
	"github.com/takoeight0821/anma/eval"
	"github.com/takoeight0821/anma/infix"
	"github.com/takoeight0821/anma/nameresolve"
	"github.com/takoeight0821/anma/token"
	"github.com/takoeight0821/anma/utils"
)

func TestEvalFromTestData(t *testing.T) {
	testcases := utils.ReadTestData()

	for _, testcase := range testcases {
		if expected, ok := testcase.Expected["eval"]; ok {
			completeEval(t, testcase.Label, testcase.Input, expected)
		} else {
			completeEval(t, testcase.Label, testcase.Input, "no expected value")
		}
	}
}

func completeEval(t *testing.T, label string, input string, expected string) {
	runner := driver.NewPassRunner()
	runner.AddPass(codata.Flat{})
	runner.AddPass(infix.NewInfixResolver())
	runner.AddPass(nameresolve.NewResolver())

	nodes, err := runner.RunSource(input)
	if err != nil {
		t.Errorf("RunSource %s returned error: %v", label, err)
		return
	}

	ev := eval.NewEvaluator()
	values := make([]eval.Value, len(nodes))

	for i, node := range nodes {
		values[i] = ev.Eval(node)
	}

	if main, ok := ev.SearchMain(); ok {
		top := token.Token{Kind: token.IDENT, Lexeme: "toplevel", Line: 0, Literal: -1}
		ret := main.Apply(top)

		actual := ret.String()
		if actual != expected {
			t.Errorf("Eval %s returned %q, expected %q", label, actual, expected)
		}
	} else {
		t.Errorf("Eval %s returned no main function", label)
	}
}
