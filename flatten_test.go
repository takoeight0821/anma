package main_test

import (
	"testing"

	"github.com/motemen/go-testutil/dataloc"
	. "github.com/takoeight0821/anma"
)

func completeFlatten(t *testing.T, input string, expected string, loc string) {
	tokens, err := Lex(input)
	if err != nil {
		t.Errorf("Lex(%q) returned error: %v at %s", input, err, loc)
	}

	p := NewParser(tokens)
	node, err := p.Parse()
	if err != nil {
		t.Errorf("Parse(%q) returned error: %v at %s", input, err, loc)
	}

	node = Flattern(node.(Expr))

	actual := node.String()
	if actual != expected {
		t.Errorf("Parse(%q) returned %q, expected %q at %s", input, actual, expected, loc)
	}
}

func TestFlatten(t *testing.T) {
	testcases := []struct {
		input    string
		expected string
	}{
		{"{ #.head -> 1, #.tail -> 2 }", "(object (field head (literal 1)) (field tail (literal 2)))"},
		{"{ #(x, y).f.g -> x + y }", "(lambda (paren (var x0) (var x1)) (object (field f (object (field g (case (paren (var x0) (var x1)) (clause (paren (var x) (var y)) (binary (var x) + (var y)))))))))"},
		{"{ #(p).h -> X, #(q).h.h -> Y, #(r).i.i -> Z }", "(lambda (paren (var x0)) (object (field h (case (paren (var x0)) (clause (paren (var p)) (var X)) (clause (paren (var q)) (object (field h (case (paren (var x0)) (clause (paren (var q)) (var Y)))))))) (field i (object (field i (case (paren (var x0)) (clause (paren (var r)) (var Z))))))))"},
		{"let fib = { #.head -> 0, #.tail.head -> 1, #.tail.tail -> zipWith(add, fib, fib.tail) }", "(let (var fib) (object (field head (literal 0)) (field tail (object (field head (literal 1)) (field tail (call (var zipWith) (var add) (var fib) (access (var fib) tail)))))))"},
		{"let prune = { #(x,t).node -> t.node, #(0,t).children -> Nil, #(x,t).children -> map(prune(x-1), t.children) }", "(let (var prune) (lambda (paren (var x0) (var x1)) (object (field node (case (paren (var x0) (var x1)) (clause (paren (var x) (var t)) (access (var t) node)))) (field children (case (paren (var x0) (var x1)) (clause (paren (literal 0) (var t)) (var Nil)) (clause (paren (var x) (var t)) (call (var map) (call (var prune) (binary (var x) - (literal 1))) (access (var t) children))))))))"},
		{"{ #(x,y)->x+y}", "(lambda (paren (var x0) (var x1)) (case (paren (var x0) (var x1)) (clause (paren (var x) (var y)) (binary (var x) + (var y)))))"},
	}
	for _, testcase := range testcases {
		completeFlatten(t, testcase.input, testcase.expected, dataloc.L(testcase.input))
	}
}
