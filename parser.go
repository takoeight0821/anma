package main

import (
	"errors"
	"fmt"
	"math"
	"strings"
)

type Expr interface {
	fmt.Stringer
}

type Integer struct {
	Token
	Value int
}

func (i Integer) String() string {
	return i.Lexeme
}

func NewInteger(t Token) Integer {
	if v, ok := t.Literal.(int); ok {
		return Integer{Token: t, Value: v}
	}
	panic(fmt.Errorf("unexpected literal type %T", t.Literal))
}

var _ Expr = Integer{}

type String struct {
	Token
	Value string
}

func (s String) String() string {
	return fmt.Sprintf("%q", s.Value)
}

func NewString(t Token) String {
	if v, ok := t.Literal.(string); ok {
		return String{Token: t, Value: v}
	}
	panic(fmt.Errorf("unexpected literal type %T", t.Literal))
}

var _ Expr = String{}

type Ident struct {
	Token
	Name string
}

func (i Ident) String() string {
	return i.Name
}

func NewIdent(t Token) Ident {
	return Ident{Token: t, Name: t.Lexeme}
}

func NewKeyword(t Token, name string) Ident {
	return Ident{Token: t, Name: name}
}

var _ Expr = Ident{}

type List struct {
	Items []Expr
}

func (l List) String() string {
	var args []fmt.Stringer
	for _, i := range l.Items {
		args = append(args, i)
	}
	return parenthesize(args...)
}

func NewList(items ...Expr) List {
	return List{Items: items}
}

var _ Expr = List{}

func parenthesize(args ...fmt.Stringer) string {
	var b strings.Builder
	b.WriteString("(")
	for i, a := range args {
		if i > 0 {
			b.WriteString(" ")
		}
		s := a.String()
		b.WriteString(s)
	}
	b.WriteString(")")

	return b.String()
}

type OpKind int

const (
	Prefix OpKind = iota
	Closed
	Postfix
	Infix
)

type Symbol interface{}

type ExprPlace struct {
	Name string
}

type ListPlace struct {
	Name      string
	Separator string
	Elem      Symbol
}

var _ Symbol = ExprPlace{}

func Match(s Symbol, t Token) bool {
	switch s := s.(type) {
	case string:
		return s == t.Lexeme
	case ExprPlace:
		return false
	default:
		panic(fmt.Errorf("unexpected symbol type %T", s))
	}
}

type Op struct {
	Kind    OpKind
	LeftBp  int
	RightBp int
	Name    string
	Symbols []Symbol
}

const (
	MAX_BP = math.MaxInt16
	MIN_BP = 0
)

func NewPrefix(name string, symbols []Symbol, rightBp int) Op {
	return Op{Kind: Prefix, LeftBp: 0, RightBp: rightBp, Name: name, Symbols: symbols}
}

func NewClosed(name string, symbols []Symbol) Op {
	return Op{Kind: Closed, LeftBp: 0, RightBp: 0, Name: name, Symbols: symbols}
}

func NewPostfix(name string, symbols []Symbol, leftBp int) Op {
	return Op{Kind: Postfix, LeftBp: leftBp, RightBp: 0, Name: name, Symbols: symbols}
}

func NewInfix(name string, symbols []Symbol, leftBp int, rightBp int) Op {
	return Op{Kind: Infix, LeftBp: leftBp, RightBp: rightBp, Name: name, Symbols: symbols}
}

type Language struct {
	leading   []Op
	following []Op
}

type Parser struct {
	lang    Language
	tokens  []Token // must be end with EOF
	current int
	err     error
}

func NewParser(tokens []Token) *Parser {
	language := Language{
		leading: []Op{
			NewPrefix("-", []Symbol{"-"}, 51),
			NewPrefix("#", []Symbol{"#"}, MIN_BP),
			NewClosed("tuple", []Symbol{"(", ListPlace{"tuple", ",", ExprPlace{"elem"}}, ")"}),
			NewClosed("func", []Symbol{"{", ExprPlace{"pattern"}, "->", ExprPlace{"body"}, "}"}),
		},
		following: []Op{
			NewPostfix("?", []Symbol{"?"}, 20),
			NewPostfix("apply", []Symbol{"(", ListPlace{"arguments", ",", ExprPlace{"argument"}}, ")"}, MAX_BP),
			NewInfix("+", []Symbol{"+"}, 50, 51),
			NewInfix("-", []Symbol{"-"}, 50, 51),
			NewInfix("*", []Symbol{"*"}, 80, 81),
		},
	}
	return &Parser{lang: language, tokens: tokens, current: 0, err: nil}
}

func (p *Parser) Parse() (result Expr, err error) {
	result = p.expr(0)
	err = p.err
	p.err = nil
	return
}

func (p Parser) peek() Token {
	return p.tokens[p.current]
}

func (p *Parser) advance() Token {
	if !p.IsAtEnd() {
		p.current++
	}
	return p.previous()
}

func parseError(t Token, message string) error {
	if t.Kind == EOF {
		return fmt.Errorf("at end: %s", message)
	}
	return fmt.Errorf("at line %d: %q: %s", t.Line, t.Lexeme, message)
}

func (p Parser) previous() Token {
	return p.tokens[p.current-1]
}

func (p Parser) IsAtEnd() bool {
	return p.peek().Kind == EOF
}

func (p *Parser) recover(err error) {
	p.err = errors.Join(p.err, err)
}

func (p *Parser) atom() Expr {
	switch t := p.advance(); t.Kind {
	case INTEGER:
		return NewInteger(t)
	case STRING:
		return NewString(t)
	case IDENT:
		return NewIdent(t)
	default:
		p.recover(fmt.Errorf("unexpected token %q", t.Lexeme))
		return nil
	}
}

func (p *Parser) symbol(s Symbol) Expr {
	switch s := s.(type) {
	case string:
		if !Match(s, p.peek()) {
			p.recover(parseError(p.peek(), fmt.Sprintf("expected %q, got %q", s, p.peek().Lexeme)))
		} else {
			p.advance()
		}
		return nil
	case ExprPlace:
		return p.expr(0)
	case ListPlace:
		var children []Expr
		for {
			child := p.symbol(s.Elem)
			if child == nil {
				break
			}
			children = append(children, child)
			if !Match(s.Separator, p.peek()) {
				break
			}
			p.advance()
		}
		return NewList(children...)
	default:
		panic(fmt.Errorf("unexpected symbol type %T", s))
	}
}

func (p *Parser) expr(minBp int) Expr {
	var lead Expr
	{
		var expr Expr
		t := p.peek()

		for _, op := range p.lang.leading {
			if Match(op.Symbols[0], t) {
				p.advance()
				children := []Expr{NewKeyword(t, op.Name)}

				for _, s := range op.Symbols[1:] {
					child := p.symbol(s)
					if child != nil {
						children = append(children, child)
					}
				}

				if op.Kind == Prefix {
					follow := p.expr(op.RightBp)
					children = append(children, follow)
				}

				expr = NewList(children...)
			}
		}

		if expr == nil {
			// No leading operator found
			lead = p.atom()
		} else {
			lead = expr
		}
	}

main:
	for {
		t := p.peek()
		if t.Kind == EOF {
			return lead
		}
		for _, op := range p.lang.following {
			if Match(op.Symbols[0], t) {
				if op.LeftBp <= minBp {
					return lead
				}

				p.advance()
				children := []Expr{NewKeyword(t, op.Name), lead}

				for _, s := range op.Symbols[1:] {
					child := p.symbol(s)
					if child != nil {
						children = append(children, child)
					}
				}

				if op.Kind == Infix {
					follow := p.expr(op.RightBp)
					children = append(children, follow)
				}

				lead = NewList(children...)
				continue main // continue to check following operators
			}
		}

		// No following operator found
		return lead
	}
}
