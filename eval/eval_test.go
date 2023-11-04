package eval_test

import (
	"testing"

	"github.com/takoeight0821/anma/ast"
	"github.com/takoeight0821/anma/codata"
	"github.com/takoeight0821/anma/driver"
	"github.com/takoeight0821/anma/eval"
	"github.com/takoeight0821/anma/infix"
	"github.com/takoeight0821/anma/rename"
)

func TestEval(t *testing.T) {
	testcases := []struct {
		input    []string
		expected string
	}{
		{[]string{"prim(add, 1, 2)"}, "3"},
		{[]string{"def + = { #(x, y) -> prim(add, x, y) }", "1 + 2"}, "3"},
	}

	for _, testcase := range testcases {
		completeEval(t, testcase.input, testcase.expected)
	}
}

func completeEval(t *testing.T, input []string, expected string) {
	runner := driver.NewPassRunner()
	runner.AddPass(codata.Flat{})
	runner.AddPass(infix.NewInfixResolver())
	runner.AddPass(rename.NewRenamer())

	var nodes []ast.Node
	for _, src := range input {
		ns, err := runner.RunSource(src)
		if err != nil {
			t.Errorf("RunSource returned error: %v", err)
		}
		nodes = append(nodes, ns...)
	}

	values := make([]eval.Value, len(nodes))

	for i, node := range nodes {
		var err error
		values[i], err = eval.NewEvaluator().Eval(node)
		if err != nil {
			t.Errorf("Eval returned error: %v", err)
		}
	}

	actual := values[len(values)-1].String()
	if actual != expected {
		t.Errorf("Eval returned %q, expected %q", actual, expected)
	}
}
