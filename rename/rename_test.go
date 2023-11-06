package rename_test

import (
	"strings"
	"testing"

	"github.com/takoeight0821/anma/codata"
	"github.com/takoeight0821/anma/driver"
	"github.com/takoeight0821/anma/infix"
	"github.com/takoeight0821/anma/rename"
)

func completeRename(t *testing.T, input, expected string) {
	runner := driver.NewPassRunner()
	runner.AddPass(codata.Flat{})
	runner.AddPass(infix.NewInfixResolver())
	runner.AddPass(rename.NewRenamer())

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
		t.Errorf("Renamer returned\n%q, expected\n%q", actual, expected)
	}
}

func TestRename(t *testing.T) {
	testcases := []struct {
		input    string
		expected string
	}{
		{"def f = { #(x) -> x }", "(def f.0 (lambda (var x0.1) (case (var x0.1) (clause (var x.2) (var x.2)))))\n"},
		{"def f = { #(f) -> f }", "(def f.0 (lambda (var x0.1) (case (var x0.1) (clause (var f.2) (var f.2)))))\n"},
		{"def + = { #(x, y) -> prim(add, x, y) }\ndef main = { #() -> 1 + 2 }\n", "(def +.0 (lambda (paren (var x0.1) (var x1.2)) (case (paren (var x0.1) (var x1.2)) (clause (paren (var x.3) (var y.4)) (prim add (var x.3) (var y.4))))))\n(def main.5 (lambda (paren) (case (paren) (clause (paren) (binary (literal 1) +.0 (literal 2))))))\n"},
	}
	for _, testcase := range testcases {
		completeRename(t, testcase.input, testcase.expected)
	}
}
