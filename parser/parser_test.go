package parser_test

import (
	"testing"

	"github.com/motemen/go-testutil/dataloc"
	"github.com/takoeight0821/anma/parser"
)

func completeParse(t *testing.T, input string, expected string, loc string) {
	tokens, err := parser.Lex(input)
	if err != nil {
		t.Errorf("Lex(%q) returned error: %v at %s", input, err, loc)
	}

	p := parser.NewParser(tokens)
	node, err := p.Parse()
	if err != nil {
		t.Errorf("Parse(%q) returned error: %v at %s", input, err, loc)
	}

	actual := node.String()
	if actual != expected {
		t.Errorf("Parse(%q) returned %q, expected %q at %s", input, actual, expected, loc)
	}
}

func TestParse(t *testing.T) {
	testcases := []struct {
		input    string
		expected string
	}{
		{"1", "(literal 1)"},
		{`"hello"`, `(literal "hello")`},
		{"f()", "(call (var f))"},
		{"f(1)", "(call (var f) (literal 1))"},
		{"f(1, 2)", "(call (var f) (literal 1) (literal 2))"},
		{"f(1)(2)", "(call (call (var f) (literal 1)) (literal 2))"},
		{"f(1,)", "(call (var f) (literal 1))"},
		{"a.b", "(access (var a) b)"},
		{"a.b.c", "(access (access (var a) b) c)"},
		{"f(x) + g(y).z", "(binary (call (var f) (var x)) + (access (call (var g) (var y)) z))"},
		{"x : Int", "(assert (var x) (var Int))"},
		{"let x = 1", "(let (var x) (literal 1))"},
		{"let x = 1 : Int", "(let (var x) (assert (literal 1) (var Int)))"},
		{"let Cons(x, xs) = list", "(let (call (var Cons) (var x) (var xs)) (var list))"},
		{"{ #(x, y) -> x + y }", "(codata (clause (call (this #) (var x) (var y)) (binary (var x) + (var y))))"},
		{"{ #(x, y) -> x + y, #(x, y) -> x - y }", "(codata (clause (call (this #) (var x) (var y)) (binary (var x) + (var y))) (clause (call (this #) (var x) (var y)) (binary (var x) - (var y))))"},
		{"fn x { x + 1 }", "(lambda (var x) (binary (var x) + (literal 1)))"},
		{"(x, y, z)", "(paren (var x) (var y) (var z))"},
		{"()", "(paren)"},
	}
	for _, testcase := range testcases {
		completeParse(t, testcase.input, testcase.expected, dataloc.L(testcase.input))
	}
}
