package main

import (
	"fmt"
	"strings"

	"golang.org/x/exp/slices"
)

func Flattern(n Expr) Expr {
	switch n := n.(type) {
	case Codata:
		return flatternCodata(n)
	case Paren:
		es := make([]Expr, len(n.Exprs))
		for i, e := range n.Exprs {
			es[i] = Flattern(e)
		}
		return Paren{es}
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
			es := make([]Expr, len(f.Exprs))
			for i, e := range f.Exprs {
				es[i] = Flattern(e)
			}
			fs[i] = Field{f.Name, es}
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
		for j, e := range cl.Exprs {
			cl.Exprs[j] = Flattern(e)
		}
		newClauses[i] = Clause{ps[i], cl.Exprs}
	}

	arity, err := Arity(ps)
	if err != nil {
		panic(err)
	}

	return NewBuilder().Build(arity, newClauses)
}

type Builder struct {
	Scrutinees []Expr
}

func NewBuilder() *Builder {
	return &Builder{}
}

// dispatch to Object or Lambda based on arity
func (b *Builder) Build(arity int, clauses []Clause) Expr {
	if arity == 0 {
		return b.Object(clauses)
	}
	return b.Lambda(arity, clauses)
}

func (b *Builder) Object(clauses []Clause) Expr {
	// Pop the first accessor of each clause and group remaining clauses by the popped accessor.
	next := make(map[string][]Clause)
	nextKeys := make([]string, 0) // for deterministic order
	for _, c := range clauses {
		plist := c.Pattern.(PatternList)
		if field, plist, ok := Pop(plist); ok {
			next[field.String()] = append(next[field.String()], Clause{plist, c.Exprs})
			if !slices.Contains(nextKeys, field.String()) {
				nextKeys = append(nextKeys, field.String())
			}
		} else {
			panic(fmt.Errorf("not implemented: %v\nmix of pure pattern and copattern is not supported yet", c))
		}
	}

	fields := make([]Field, 0)

	// Generate each field's body expression
	for _, field := range nextKeys {
		cs := next[field]

		allHasAccessors := true
		for _, c := range cs {
			if len(c.Pattern.(PatternList).Accessors) == 0 {
				allHasAccessors = false
				break
			}
		}
		if allHasAccessors {
			// if all pattern lists have accessors, call Object recursively
			fields = append(fields, Field{field, []Expr{b.Object(cs)}})
		} else if len(b.Scrutinees) != 0 {
			// if any of cs has no accessors and has guards, generate Case expression
			caseClauses := make([]Clause, 0)
			restClauses := make([]Clause, 0)
			for _, c := range cs {
				plist := c.Pattern.(PatternList)
				if len(plist.Accessors) == 0 {
					caseClauses = append(caseClauses, Clause{Paren{plist.Params}, c.Exprs})
				} else {
					restClauses = append(restClauses, c)
				}
			}

			for _, c := range restClauses {
				plist := c.Pattern.(PatternList)
				caseClauses = append(caseClauses, Clause{Paren{plist.Params}, []Expr{b.Object(restClauses)}})
			}
			fields = append(fields, Field{field, []Expr{Case{Paren{b.Scrutinees}, caseClauses}}})
		} else {
			// if there is no scrutinee, simply insert the clause's body expression
			fields = append(fields, Field{field, cs[0].Exprs})
		}
	}
	return Object{fields}
}

// Generate Lambda and dispatch body expression to Object or Case based on existence of accessors.
func (b *Builder) Lambda(arity int, clauses []Clause) Expr {
	baseToken := Codata{clauses}.Base()
	// Generate Scrutinees
	b.Scrutinees = make([]Expr, arity)
	for i := 0; i < arity; i++ {
		b.Scrutinees[i] = Var{Token{IDENT, fmt.Sprintf("x%d", i), baseToken.Line, nil}}
	}

	// If any of clauses has accessors, body expression is Object.
	for _, c := range clauses {
		if len(c.Pattern.(PatternList).Accessors) != 0 {
			return Lambda{Pattern: Paren{b.Scrutinees}, Exprs: []Expr{b.Object(clauses)}}
		}
	}

	// otherwise, body expression is Case.
	caseClauses := make([]Clause, 0)
	for _, c := range clauses {
		plist := c.Pattern.(PatternList)
		caseClauses = append(caseClauses, Clause{Paren{plist.Params}, c.Exprs})
	}
	return Lambda{Pattern: Paren{b.Scrutinees}, Exprs: []Expr{Case{Paren{b.Scrutinees}, caseClauses}}}
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
		return p.Args
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

// Split PatternList into the first accessor and the rest.
func Pop(p PatternList) (Token, PatternList, bool) {
	if len(p.Accessors) == 0 {
		return Token{}, p, false
	}
	a := p.Accessors[0]
	p.Accessors = p.Accessors[1:]
	return a, p, true
}
