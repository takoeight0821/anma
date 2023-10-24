package main_test

import (
	"strings"
	"testing"

	. "github.com/takoeight0821/anma"
)

func completeRename(t *testing.T, input, expected string) {
	tokens, err := Lex(input)
	if err != nil {
		t.Errorf("Lex returned error: %v", err)
	}

	nodes, err := NewParser(tokens).ParseDecl()
	if err != nil {
		t.Errorf("Parse returned error: %v", err)
	}

	for i, node := range nodes {
		nodes[i] = Flat(node)
	}

	infix := NewInfixResolver()
	for _, node := range nodes {
		infix.Load(node)
	}

	rename := NewRenamer()

	for i, node := range nodes {
		nodes[i] = rename.Solve(infix.Resolve(Flat(node)))
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