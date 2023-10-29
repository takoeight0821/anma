package codata_test

import (
	"strings"
	"testing"

	"github.com/takoeight0821/anma/codata"
	"github.com/takoeight0821/anma/driver"
)

func completeFlat(t *testing.T, input string, expected string) {
	runner := driver.NewPassRunner()
	runner.AddPass(codata.Flat{})
	node, err := runner.RunSource(input)
	if err != nil {
		t.Errorf("RunSource returned error: %v", err)
	}

	actual := node[0].String()
	if actual != expected {
		t.Errorf("Flat returned\n%q, expected\n%q", actual, expected)
	}
}

func TestFlat(t *testing.T) {
	testcases := []struct {
		input    string
		expected string
	}{
		{"{ #.head -> 1, #.tail -> 2 }", "(object (field head (literal 1)) (field tail (literal 2)))"},
		{"{ #(x, y).f.g -> x + y }", "(lambda (paren (var x0) (var x1)) (object (field f (object (field g (case (paren (var x0) (var x1)) (clause (paren (var x) (var y)) (binary (var x) + (var y)))))))))"},
		{"{ #(p).h -> X, #(q).h.h -> Y, #(r).i.i -> Z }", "(lambda (paren (var x0)) (object (field h (case (paren (var x0)) (clause (paren (var p)) (var X)) (clause (paren (var q)) (object (field h (case (paren (var x0)) (clause (paren (var q)) (var Y)))))))) (field i (object (field i (case (paren (var x0)) (clause (paren (var r)) (var Z))))))))"},
		{"let fib = { #.head -> 0, #.tail.head -> 1, #.tail.tail -> zipWith(add, fib, fib.tail) }", "(let (var fib) (object (field head (literal 0)) (field tail (object (field head (literal 1)) (field tail (call (var zipWith) (var add) (var fib) (access (var fib) tail)))))))"},
		{"let prune = { #(x,t).node -> t.node, #(0,t).children -> Nil, #(x,t).children -> map(prune(x-1), t.children) }", "(let (var prune) (lambda (paren (var x0) (var x1)) (object (field children (case (paren (var x0) (var x1)) (clause (paren (literal 0) (var t)) (var Nil)) (clause (paren (var x) (var t)) (call (var map) (call (var prune) (binary (var x) - (literal 1))) (access (var t) children))))) (field node (case (paren (var x0) (var x1)) (clause (paren (var x) (var t)) (access (var t) node)))))))"},
		{"{ #(x,y)->x+y}", "(lambda (paren (var x0) (var x1)) (case (paren (var x0) (var x1)) (clause (paren (var x) (var y)) (binary (var x) + (var y)))))"},
	}
	for _, testcase := range testcases {
		completeFlat(t, testcase.input, testcase.expected)
	}
}

func completeFlatDecl(t *testing.T, input string, expected string) {
	runner := driver.NewPassRunner()
	runner.AddPass(codata.Flat{})

	node, err := runner.RunSource(input)
	if err != nil {
		t.Errorf("RunSource returned error: %v", err)
	}

	var b strings.Builder
	for _, decl := range node {
		b.WriteString(decl.String())
		b.WriteString("\n")
	}
	actual := b.String()
	if actual != expected {
		t.Errorf("Flat returned\n%q, expected\n%q", actual, expected)
	}
}

func TestFlatDecl(t *testing.T) {
	testcases := []struct {
		input    string
		expected string
	}{
		{"type List = { head: Int, tail: List }", "(type List (object (field head (var Int)) (field tail (var List))))\n"},
		{"def fib = { #.head -> 0, #.tail.head -> 1, #.tail.tail -> zipWith(add, fib, fib.tail) }", "(def fib (object (field head (literal 0)) (field tail (object (field head (literal 1)) (field tail (call (var zipWith) (var add) (var fib) (access (var fib) tail)))))))\n"},
	}
	for _, testcase := range testcases {
		completeFlatDecl(t, testcase.input, testcase.expected)
	}
}
