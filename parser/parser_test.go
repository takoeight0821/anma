package parser_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/takoeight0821/anma/driver"
	"github.com/takoeight0821/anma/lexer"
	"github.com/takoeight0821/anma/parser"
	"github.com/takoeight0821/anma/utils"
)

func completeParseExpr(t *testing.T, input string, expected string) {
	tokens, err := lexer.Lex(input)
	if err != nil {
		t.Errorf("Lex(%q) returned error: %v", input, err)
	}

	p := parser.NewParser(tokens)
	node, err := p.ParseExpr()
	if err != nil {
		t.Errorf("Parse(%q) returned error: %v", input, err)
	}

	actual := node.String()
	if actual != expected {
		t.Errorf("Parse(%q) returned %q, expected %q", input, actual, expected)
	}
}

var testcases = []struct {
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
	{"{ #(x, y) -> x + y; x }", "(codata (clause (call (this #) (var x) (var y)) (binary (var x) + (var y)) (var x)))"},
	{"{ #(x, y) -> x + y; x; }", "(codata (clause (call (this #) (var x) (var y)) (binary (var x) + (var y)) (var x)))"},
	{"{ #(x, y) -> x + y, #(x, y) -> x - y }", "(codata (clause (call (this #) (var x) (var y)) (binary (var x) + (var y))) (clause (call (this #) (var x) (var y)) (binary (var x) - (var y))))"},
	{"{ #(x, y) -> x + y, #(x, y) -> x - y, }", "(codata (clause (call (this #) (var x) (var y)) (binary (var x) + (var y))) (clause (call (this #) (var x) (var y)) (binary (var x) - (var y))))"},
	{"fn x { x + 1 }", "(lambda (var x) (binary (var x) + (literal 1)))"},
	{"(x, y, z)", "(paren (var x) (var y) (var z))"},
	{"(x, y, z,)", "(paren (var x) (var y) (var z))"},
	{"()", "(paren)"},
	{"f : a -> b", "(assert (var f) (binary (var a) -> (var b)))"},
	{"fn x { let y = 1; x + y }", "(lambda (var x) (let (var y) (literal 1)) (binary (var x) + (var y)))"},
	{"fn x { let y = 1; x + y; }", "(lambda (var x) (let (var y) (literal 1)) (binary (var x) + (var y)))"},
	{"{ #.head -> 1 }", "(codata (clause (access (this #) head) (literal 1)))"},
	{"prim(add, 1, 2)", "(prim add (literal 1) (literal 2))"},
}

func TestParse(t *testing.T) {
	for _, testcase := range testcases {
		completeParseExpr(t, testcase.input, testcase.expected)
	}
}

func completeParseDecl(t *testing.T, input string, expected string) {
	tokens, err := lexer.Lex(input)
	if err != nil {
		t.Errorf("Lex(%q) returned error: %v", input, err)
	}

	p := parser.NewParser(tokens)
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

var testcasesDecl = []struct {
	input    string
	expected string
}{
	{"def x = 1", "(def x (literal 1))\n"},
	{"type List = { head: Int, tail: List }", "(type List (object (field head (var Int)) (field tail (var List))))\n"},
}

func TestParseDecl(t *testing.T) {
	for _, testcase := range testcasesDecl {
		completeParseDecl(t, testcase.input, testcase.expected)
	}
}

func tryParse(t *testing.T, input string) {
	tokens, err := lexer.Lex(input)
	if err != nil {
		t.Logf("Lex(%q) returned error: %v", input, err)
		return
	}
	t.Logf("tokens: %v", tokens)

	p := parser.NewParser(tokens)
	_, err = p.ParseExpr()
	if err != nil {
		t.Logf("Parse(%q) returned error: %v", input, err)
		return
	}

	p = parser.NewParser(tokens)
	_, err = p.ParseDecl()
	if err != nil {
		t.Logf("Parse(%q) returned error: %v", input, err)
		return
	}
}

func TestParseFromTestData(t *testing.T) {
	testcases := utils.ReadTestData()
	for _, testcase := range testcases {
		if expected, ok := testcase.Expected["parser"]; ok {
			newCompleteParse(t, testcase.Input, expected)
		} else {
			t.Logf("no expected result for %q", testcase.Input)
		}
	}
}

func newCompleteParse(t *testing.T, input string, expected string) {
	r := driver.NewPassRunner()

	nodes, err := r.RunSource(input)
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
		t.Errorf("RunSource returned %q, expected %q", actual, expected)
	}
}

// parseExpr must stop for any random string.
// go test -fuzz=Fuzz -fuzztime 1000x
func FuzzParseExpr(f *testing.F) {
	for _, testcase := range testcases {
		f.Add(testcase.input)
	}
	for _, testcase := range testcasesDecl {
		f.Add(testcase.input)
	}
	f.Fuzz(func(t *testing.T, input string) {
		if utf8.ValidString(input) {
			t.Logf("input: %q", input)
			tryParse(t, input)
		}
	})
}
