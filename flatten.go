package main

import (
	"fmt"
	"strings"

	"github.com/takoeight0821/anma/ast"
	"github.com/takoeight0821/anma/token"
	"golang.org/x/exp/slices"
)

func Flattern(n ast.Node) ast.Node {
	return ast.Traverse(n, flattern, ast.Any)
}

func flattern(n ast.Node, k ast.Kind) ast.Node {
	if n, ok := n.(ast.Codata); ok {
		return flatternCodata(n)
	}
	return n
}

func flatternCodata(c ast.Codata) ast.Node {
	// Generate PatternList
	arity := -1
	for i, cl := range c.Clauses {
		c.Clauses[i].Pattern = PatternList{Accessors: accessors(cl.Pattern), Params: params(cl.Pattern)}
		if arity == -1 {
			arity = len(c.Clauses[i].Pattern.(PatternList).Params)
		} else if arity != len(c.Clauses[i].Pattern.(PatternList).Params) {
			panic(fmt.Errorf("arity mismatch at %d: %v", c.Base().Line, c))
		}
	}

	if arity == -1 {
		panic(fmt.Errorf("unreachable: arity is -1 at %d: %v", c.Base().Line, c))
	}

	return NewBuilder().Build(arity, c.Clauses)
}

type Builder struct {
	Scrutinees []ast.Node
}

func NewBuilder() *Builder {
	return &Builder{}
}

// dispatch to Object or Lambda based on arity
func (b *Builder) Build(arity int, clauses []ast.Clause) ast.Node {
	if arity == 0 {
		return b.Object(clauses)
	}
	return b.Lambda(arity, clauses)
}

func (b *Builder) Object(clauses []ast.Clause) ast.Object {
	// Pop the first accessor of each clause and group remaining clauses by the popped accessor.
	next := make(map[string][]ast.Clause)
	nextKeys := make([]string, 0) // for deterministic order
	for _, c := range clauses {
		plist := c.Pattern.(PatternList)
		if field, plist, ok := Pop(plist); ok {
			next[field.String()] = append(next[field.String()], ast.Clause{Pattern: plist, Exprs: c.Exprs})
			if !slices.Contains(nextKeys, field.String()) {
				nextKeys = append(nextKeys, field.String())
			}
		} else {
			panic(fmt.Errorf("not implemented: %v\nmix of pure pattern and copattern is not supported yet", c))
		}
	}

	fields := make([]ast.Field, 0)

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
			fields = append(fields, ast.Field{Name: field, Exprs: []ast.Node{b.Object(cs)}})
		} else if len(b.Scrutinees) != 0 {
			// if any of cs has no accessors and has guards, generate Case expression
			caseClauses := make([]ast.Clause, 0)
			restClauses := make([]ast.Clause, 0)
			for _, c := range cs {
				plist := c.Pattern.(PatternList)
				if len(plist.Accessors) == 0 {
					caseClauses = append(caseClauses, ast.Clause{Pattern: ast.Paren{Elems: plist.Params}, Exprs: c.Exprs})
				} else {
					restClauses = append(restClauses, c)
				}
			}

			for _, c := range restClauses {
				plist := c.Pattern.(PatternList)
				caseClauses = append(caseClauses, ast.Clause{Pattern: ast.Paren{Elems: plist.Params}, Exprs: []ast.Node{b.Object(restClauses)}})
			}
			fields = append(fields, ast.Field{Name: field, Exprs: []ast.Node{ast.Case{Scrutinee: ast.Paren{Elems: b.Scrutinees}, Clauses: caseClauses}}})
		} else {
			// if there is no scrutinee, simply insert the clause's body expression
			fields = append(fields, ast.Field{Name: field, Exprs: cs[0].Exprs})
		}
	}
	return ast.Object{Fields: fields}
}

// Generate Lambda and dispatch body expression to Object or Case based on existence of accessors.
func (b *Builder) Lambda(arity int, clauses []ast.Clause) ast.Lambda {
	baseToken := ast.Codata{Clauses: clauses}.Base()
	// Generate Scrutinees
	b.Scrutinees = make([]ast.Node, arity)
	for i := 0; i < arity; i++ {
		b.Scrutinees[i] = ast.Var{Name: token.Token{Kind: token.IDENT, Lexeme: fmt.Sprintf("x%d", i), Line: baseToken.Line, Literal: nil}}
	}

	// If any of clauses has accessors, body expression is Object.
	for _, c := range clauses {
		if len(c.Pattern.(PatternList).Accessors) != 0 {
			return ast.Lambda{Pattern: ast.Paren{Elems: b.Scrutinees}, Exprs: []ast.Node{b.Object(clauses)}}
		}
	}

	// otherwise, body expression is Case.
	caseClauses := make([]ast.Clause, 0)
	for _, c := range clauses {
		plist := c.Pattern.(PatternList)
		caseClauses = append(caseClauses, ast.Clause{Pattern: ast.Paren{Elems: plist.Params}, Exprs: c.Exprs})
	}
	return ast.Lambda{Pattern: ast.Paren{Elems: b.Scrutinees}, Exprs: []ast.Node{ast.Case{Scrutinee: ast.Paren{Elems: b.Scrutinees}, Clauses: caseClauses}}}
}

func InvalidPattern(n ast.Node) error {
	return fmt.Errorf("invalid pattern: %v", n)
}

// Collect all Access patterns recursively.
func accessors(p ast.Node) []token.Token {
	switch p := p.(type) {
	case ast.Access:
		return append(accessors(p.Receiver), p.Name)
	default:
		return []token.Token{}
	}
}

// Get Args of Call{This, ...}
func params(p ast.Node) []ast.Node {
	switch p := p.(type) {
	case ast.Access:
		return params(p.Receiver)
	case ast.Call:
		if _, ok := p.Func.(ast.This); !ok {
			panic(InvalidPattern(p))
		}
		return p.Args
	default:
		return []ast.Node{}
	}
}

type PatternList struct {
	Accessors []token.Token
	Params    []ast.Node
}

func (p PatternList) Base() token.Token {
	if len(p.Accessors) == 0 {
		if len(p.Params) == 0 {
			return token.Token{}
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

var _ ast.Node = PatternList{}

// Split PatternList into the first accessor and the rest.
func Pop(p PatternList) (token.Token, PatternList, bool) {
	if len(p.Accessors) == 0 {
		return token.Token{}, p, false
	}
	a := p.Accessors[0]
	p.Accessors = p.Accessors[1:]
	return a, p, true
}
