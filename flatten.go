package main

import (
	"fmt"
	"strings"
)

func Flattern(n Expr) Expr {
	switch n := n.(type) {
	case Codata:
		return flatternCodata(n)
	case Paren:
		return Paren{Flattern(n.Expr)}
	case Access:
		return Access{Flattern(n.Expr), n.Name}
	case Call:
		as := make([]Expr, len(n.Args))
		for i, a := range n.Args {
			as[i] = Flattern(a)
		}
		return Call{Flattern(n.Func), as}
	case Binary:
		return Binary{Flattern(n.Left), n.Op, Flattern(n.Right)}
	case Assert:
		return Assert{Flattern(n.Expr), n.Type}
	case Let:
		return Let{Flattern(n.Pattern), Flattern(n.Expr)}
	case Lambda:
		es := make([]Expr, len(n.Exprs))
		for i, e := range n.Exprs {
			es[i] = Flattern(e)
		}
		return Lambda{Flattern(n.Pattern), es}
	case Case:
		cs := make([]Clause, len(n.Clauses))
		for i, c := range n.Clauses {
			es := make([]Expr, len(c.Exprs))
			for i, e := range c.Exprs {
				es[i] = Flattern(e)
			}
			cs[i] = Clause{Flattern(c.Pattern), es}
		}
		return Case{Flattern(n.Expr), cs}
	case Object:
		fs := make([]Field, len(n.Fields))
		for i, f := range n.Fields {
			fs[i] = Field{f.Name, Flattern(f.Expr)}
		}
		return Object{fs}
	default:
		return n
	}
}

func flatternCodata(c Codata) Expr {
	// Generate PatternList
	ps := make([]PatternList, len(c.Clauses))
	for i, cl := range c.Clauses {
		as := accessors(cl.Pattern)
		p := params(cl.Pattern)
		ps[i] = PatternList{as, p}
	}

	newClauses := make([]Clause, len(c.Clauses))
	for i, cl := range c.Clauses {
		newClauses[i] = Clause{ps[i], cl.Exprs}
	}

	arity, err := Arity(ps)
	if err != nil {
		panic(err)
	}

	return NewBuilder().Build(arity, newClauses)
}

type Builder struct{}

func NewBuilder() *Builder {
	return &Builder{}
}

func (b *Builder) Build(arity int, clauses []Clause) Expr {
	if arity == 0 {
		return b.Object(clauses)
	}
	return b.Lambda(arity, clauses)
}

func (b *Builder) Object(clauses []Clause) Expr {
	panic("Not implemented")
}

func (b *Builder) Lambda(arity int, clauses []Clause) Expr {
	panic("Not implemented")
}

func InvalidPattern(n Node) error {
	return fmt.Errorf("invalid pattern: %v", n)
}

// Collect all Access patterns recursively.
func accessors(p Pattern) []Token {
	switch p := p.(type) {
	case Access:
		return append(accessors(p.Expr), p.Name)
	default:
		return []Token{}
	}
}

// Get Args of Call{This, ...}
func params(p Pattern) []Pattern {
	switch p := p.(type) {
	case Access:
		return params(p.Expr)
	case Call:
		if _, ok := p.Func.(This); !ok {
			panic(InvalidPattern(p))
		}
		ps := make([]Pattern, len(p.Args))
		for i, a := range p.Args {
			ps[i] = a.(Pattern)
		}
		return ps
	default:
		return []Pattern{}
	}
}

type PatternList struct {
	Accessors []Token
	Params    []Pattern
}

func (p PatternList) Base() Token {
	if len(p.Accessors) == 0 {
		if len(p.Params) == 0 {
			return Token{}
		}
		return p.Params[0].Base()
	}
	return p.Accessors[0]
}

func (p PatternList) String() string {
	var b strings.Builder
	b.WriteString("[")
	for i, a := range p.Accessors {
		if i > 0 {
			b.WriteString(" ")
		}
		b.WriteString(a.String())
	}
	b.WriteString(" | ")
	for i, p := range p.Params {
		if i > 0 {
			b.WriteString(" ")
		}
		b.WriteString(p.String())
	}
	b.WriteString("]")
	return b.String()
}

func (p PatternList) ValidPattern() bool {
	b := true
	for _, p := range p.Params {
		b = b && p.ValidPattern()
	}
	return b
}

func (p PatternList) ValidType() bool {
	return false
}

var _ Pattern = PatternList{}

// Returns the length of every Params in the list.
func Arity(ps []PatternList) (int, error) {
	if len(ps) == 0 {
		return 0, fmt.Errorf("empty pattern list")
	}

	arity := len(ps[0].Params)
	for _, p := range ps {
		if len(p.Params) != arity {
			return 0, fmt.Errorf("invalid arity at %v: %d", p, arity)
		}
	}

	return arity, nil
}

// Returns the last element of Accessors and removes it.
func Pop(p *PatternList) (Token, bool) {
	if len(p.Accessors) == 0 {
		return Token{}, false
	}
	a := p.Accessors[len(p.Accessors)-1]
	p.Accessors = p.Accessors[:len(p.Accessors)-1]
	return a, true
}
