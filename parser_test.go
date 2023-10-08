package main_test

import (
	"errors"
	"fmt"
	"testing"

	. "github.com/takoeight0821/tenchi"
)

func TestExprPrint(t *testing.T) {
	expr := List{
		Items: []Expr{
			Ident{
				Token: Token{IDENT, "add", 1, nil},
				Name:  "add",
			},
			Integer{
				Token: Token{INTEGER, "123", 1, 123},
				Value: 123,
			},
			String{
				Token: Token{STRING, `"hello"`, 1, "hello"},
				Value: "hello",
			},
		},
	}

	expected := "(add 123 \"hello\")"
	actual := expr.String()
	if actual != expected {
		t.Errorf("expected %q, got %q", expected, actual)
	}
}

func completeParse(t *testing.T, input string, expected string) {
	tokens, err := Lex(input)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	p := NewParser(tokens)
	expr, err := p.Parse()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if !p.IsAtEnd() {
		err = fmt.Errorf("expected to reach EOF, but not")
	}

	actual := expr.String()
	if actual != expected {
		err = errors.Join(err, fmt.Errorf("expected %q, got %q", expected, actual))
		t.Error(err)
		return
	}
}

func TestAtom(t *testing.T) {
	completeParse(t, "123", "123")
	completeParse(t, `"hello"`, `"hello"`)
	completeParse(t, "add", "add")
}

func TestSimplePrefix(t *testing.T) {
	completeParse(t, "-8", "(- 8)")
}

func TestParen(t *testing.T) {
	completeParse(t, "(-1)", "(paren (- 1))")
}

func TestSimplePostfix(t *testing.T) {
	completeParse(t, "1?", "(? 1)")
}

func TestSimpleInfix(t *testing.T) {
	completeParse(t, "1+2", "(+ 1 2)")
}

func TestInfixAndPrefix(t *testing.T) {
	completeParse(t, "1 + -2", "(+ 1 (- 2))")
}

func TestDifferentPosition(t *testing.T) {
	completeParse(t, "1 - -2", "(- 1 (- 2))")
}
