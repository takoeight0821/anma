package main_test

import (
	"strings"
	"testing"

	. "github.com/takoeight0821/anma"
)

func completeRename(t *testing.T, input, expected string) {
	runner := NewPassRunner()
	runner.AddPass(Flat{})
	runner.AddPass(NewInfixResolver())
	runner.AddPass(NewRenamer())

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
	}
	for _, testcase := range testcases {
		completeRename(t, testcase.input, testcase.expected)
	}
}
