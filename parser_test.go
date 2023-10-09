package main_test

import (
	"testing"

	. "github.com/takoeight0821/tenchi"
)

func completeParse(t *testing.T, input string, expected string) {
	tokens, err := Lex(input)
	if err != nil {
		t.Errorf("Lex(%q) returned error: %v", input, err)
	}

	p := NewParser(tokens)
	node, err := p.Parse()
	if err != nil {
		t.Errorf("Parse(%q) returned error: %v", input, err)
	}

	actual := node.String()
	if actual != expected {
		t.Errorf("Parse(%q) returned %q, expected %q", input, actual, expected)
	}
}

func TestParse(t *testing.T) {
	completeParse(t, "1", "(literal 1)")
	completeParse(t, `"hello"`, `(literal "hello")`)
	completeParse(t, "f()", "(call (var f))")
	completeParse(t, "f(1)", "(call (var f) (literal 1))")
	completeParse(t, "f(1, 2)", "(call (var f) (literal 1) (literal 2))")
	completeParse(t, "f(1)(2)", "(call (call (var f) (literal 1)) (literal 2))")
	completeParse(t, "f(1,)", "(call (var f) (literal 1))")
	completeParse(t, "a.b", "(access (var a) b)")
	completeParse(t, "a.b.c", "(access (access (var a) b) c)")
	completeParse(t, "f(x) + g(y).z", "(binary (call (var f) (var x)) + (access (call (var g) (var y)) z))")
	completeParse(t, "x : Int", "(assert (var x) (var Int))")
	completeParse(t, "let x = 1", "(let (var x) (literal 1))")
	completeParse(t, "let x = 1 : Int", "(let (var x) (assert (literal 1) (var Int)))")
	completeParse(t, "let Cons(x, xs) = list", "(let (call (var Cons) (var x) (var xs)) (var list))")
	completeParse(t, "{ #(x, y) -> x + y }", "(codata (clause (call (this #) (var x) (var y)) (binary (var x) + (var y))))")
	completeParse(t, "{ #(x, y) -> x + y, #(x, y) -> x - y }", "(codata (clause (call (this #) (var x) (var y)) (binary (var x) + (var y))) (clause (call (this #) (var x) (var y)) (binary (var x) - (var y))))")
	completeParse(t, "fn x { x + 1 }", "(lambda (var x) (binary (var x) + (literal 1)))")
}
