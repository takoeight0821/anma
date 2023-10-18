package main_test

import (
	"testing"

	. "github.com/takoeight0821/anma"
)

func completeInfix(t *testing.T, input1, input2, expected string) {
	tokens, err := Lex(input1)
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

	r := NewInfixResolver()
	for _, node := range nodes {
		r.Load(node)
	}

	tokens, err = Lex(input2)
	if err != nil {
		t.Errorf("Lex returned error: %v", err)
	}

	node, err := NewParser(tokens).ParseExpr()
	if err != nil {
		t.Errorf("Parse returned error: %v", err)
	}

	node = r.Resolve(Flat(node))

	actual := node.String()

	if actual != expected {
		t.Errorf("InfixResolver returned\n%q, expected\n%q", actual, expected)
	}
}

func TestInfix(t *testing.T) {
	testcases := []struct {
		input1   string
		input2   string
		expected string
	}{
		{"infixl 6 +\ninfixl 8 *", "1 + 2 * 3", "(binary (literal 1) + (binary (literal 2) * (literal 3)))"},
		{"infixl 6 +\ninfixl 8 *", "1 * 2 + 3", "(binary (binary (literal 1) * (literal 2)) + (literal 3))"},
	}
	for _, testcase := range testcases {
		completeInfix(t, testcase.input1, testcase.input2, testcase.expected)
	}
}
