package main_test

import (
	"testing"

	"github.com/motemen/go-testutil/dataloc"
	. "github.com/takoeight0821/tenchi"
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
		{"{ #(x, y).f.g -> x + y, #(x, y) -> x - y }", "(codata (clause [f g | (var x) (var y)] (binary (var x) + (var y))) (clause [ | (var x) (var y)] (binary (var x) - (var y))))"},
	}
	for _, testcase := range testcases {
		completeFlatten(t, testcase.input, testcase.expected, dataloc.L(testcase.input))
	}
}
