package main_test

import (
	"strings"
	"testing"

	. "github.com/takoeight0821/anma"
)

func completeParseExpr(t *testing.T, input string, expected string) {
	tokens, err := Lex(input)
	if err != nil {
		t.Errorf("Lex(%q) returned error: %v", input, err)
	}

	p := NewParser(tokens)
	node, err := p.ParseExpr()
	if err != nil {
		t.Errorf("Parse(%q) returned error: %v", input, err)
	}

	actual := node.String()
	if actual != expected {
		t.Errorf("Parse(%q) returned %q, expected %q", input, actual, expected)
	}
}

func TestParse(t *testing.T) {
	completeParseExpr(t, "1", "(literal 1)")
	completeParseExpr(t, `"hello"`, `(literal "hello")`)
	completeParseExpr(t, "f()", "(call (var f))")
	completeParseExpr(t, "f(1)", "(call (var f) (literal 1))")
	completeParseExpr(t, "f(1, 2)", "(call (var f) (literal 1) (literal 2))")
	completeParseExpr(t, "f(1)(2)", "(call (call (var f) (literal 1)) (literal 2))")
	completeParseExpr(t, "f(1,)", "(call (var f) (literal 1))")
	completeParseExpr(t, "a.b", "(access (var a) b)")
	completeParseExpr(t, "a.b.c", "(access (access (var a) b) c)")
	completeParseExpr(t, "f(x) + g(y).z", "(binary (call (var f) (var x)) + (access (call (var g) (var y)) z))")
	completeParseExpr(t, "x : Int", "(assert (var x) (var Int))")
	completeParseExpr(t, "let x = 1", "(let (var x) (literal 1))")
	completeParseExpr(t, "let x = 1 : Int", "(let (var x) (assert (literal 1) (var Int)))")
	completeParseExpr(t, "let Cons(x, xs) = list", "(let (call (var Cons) (var x) (var xs)) (var list))")
	completeParseExpr(t, "{ #(x, y) -> x + y }", "(codata (clause (call (this #) (var x) (var y)) (binary (var x) + (var y))))")
	completeParseExpr(t, "{ #(x, y) -> x + y; x }", "(codata (clause (call (this #) (var x) (var y)) (binary (var x) + (var y)) (var x)))")
	completeParseExpr(t, "{ #(x, y) -> x + y; x; }", "(codata (clause (call (this #) (var x) (var y)) (binary (var x) + (var y)) (var x)))")
	completeParseExpr(t, "{ #(x, y) -> x + y, #(x, y) -> x - y }", "(codata (clause (call (this #) (var x) (var y)) (binary (var x) + (var y))) (clause (call (this #) (var x) (var y)) (binary (var x) - (var y))))")
	completeParseExpr(t, "{ #(x, y) -> x + y, #(x, y) -> x - y, }", "(codata (clause (call (this #) (var x) (var y)) (binary (var x) + (var y))) (clause (call (this #) (var x) (var y)) (binary (var x) - (var y))))")
	completeParseExpr(t, "fn x { x + 1 }", "(lambda (var x) (binary (var x) + (literal 1)))")
	completeParseExpr(t, "(x, y, z)", "(paren (var x) (var y) (var z))")
	completeParseExpr(t, "(x, y, z,)", "(paren (var x) (var y) (var z))")
	completeParseExpr(t, "()", "(paren)")
	completeParseExpr(t, "f : a -> b", "(assert (var f) (binary (var a) -> (var b)))")
	completeParseExpr(t, "fn x { let y = 1; x + y }", "(lambda (var x) (let (var y) (literal 1)) (binary (var x) + (var y)))")
	completeParseExpr(t, "fn x { let y = 1; x + y; }", "(lambda (var x) (let (var y) (literal 1)) (binary (var x) + (var y)))")
	completeParseExpr(t, "{ #.head -> 1 }", "(codata (clause (access (this #) head) (literal 1)))")
}

func completeParseDecl(t *testing.T, input string, expected string) {
	tokens, err := Lex(input)
	if err != nil {
		t.Errorf("Lex(%q) returned error: %v", input, err)
	}

	p := NewParser(tokens)
	node, err := p.ParseDecl()
	if err != nil {
		t.Errorf("Parse(%q) returned error: %v", input, err)
	}

	var b strings.Builder
	for _, decl := range node {
		b.WriteString(decl.String())
		b.WriteString("\n")
	}

	actual := b.String()
	if actual != expected {
		t.Errorf("Parse(%q) returned %q, expected %q", input, actual, expected)
	}
}

func TestParseDecl(t *testing.T) {
	completeParseDecl(t, "def x = 1", "(def x (literal 1))\n")
	completeParseDecl(t, "type List = { head: Int, tail: List }", "(type List (object (field head (var Int)) (field tail (var List))))\n")
}
