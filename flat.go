package main

import (
	"fmt"
	"strings"

	"github.com/takoeight0821/anma/internal/ast"
	"github.com/takoeight0821/anma/internal/token"
	"github.com/takoeight0821/anma/internal/utils"
)

// Flat converts Copatterns ([Access] and [This] in [Pattern]) into [Object] and [Lambda].
type Flat struct{}

func (Flat) Init([]ast.Node) error {
	return nil
}

func (Flat) Run(program []ast.Node) ([]ast.Node, error) {
	for i, n := range program {
		program[i] = flat(n)
	}
	return program, nil
}

func flat(n ast.Node) ast.Node {
	return ast.Transform(n, flatEach)
}

func flatEach(n ast.Node) ast.Node {
	if n, ok := n.(*ast.Codata); ok {
		return flatCodata(n)
	}
	return n
}

func flatCodata(c *ast.Codata) ast.Node {
	// Generate PatternList
	arity := -1
	for i, cl := range c.Clauses {
		plist := patternList{accessors: accessors(cl.Pattern), params: params(cl.Pattern)}
		c.Clauses[i] = &ast.Clause{Pattern: plist, Exprs: cl.Exprs}
		if arity == -1 {
			arity = len(plist.params)
		} else if arity != len(plist.params) {
			panic(utils.ErrorAt(c.Base(), fmt.Sprintf("arity mismatch %v", c)))
		}
	}

	if arity == -1 {
		panic(utils.ErrorAt(c.Base(), fmt.Sprintf("unreachable: arity is -1 %v", c)))
	}

	return newBuilder().build(arity, c.Clauses)
}

type builder struct {
	scrutinees []ast.Node
}

func newBuilder() *builder {
	return &builder{}
}

// dispatch to Object or Lambda based on arity
func (b *builder) build(arity int, clauses []*ast.Clause) ast.Node {
	if arity == 0 {
		return b.object(clauses)
	}
	return b.lambda(arity, clauses)
}

func (b builder) object(clauses []*ast.Clause) ast.Node {
	// Pop the first accessor of each clause and group remaining clauses by the popped accessor.
	next := make(map[string][]*ast.Clause)
	for _, c := range clauses {
		plist := c.Pattern.(patternList)
		if field, plist, ok := pop(plist); ok {
			next[field.String()] = append(
				next[field.String()],
				&ast.Clause{Pattern: plist, Exprs: c.Exprs})
		} else {
			panic(utils.ErrorAt(c.Base(), fmt.Sprintf("not implemented: %v\nmix of pure pattern and copattern is not supported yet", c)))
		}
	}

	fields := make([]*ast.Field, 0)

	// Generate each field's body expression
	// Object fields are generated in the dictionary order of field names.
	utils.OrderedFor(next, func(field string, cs []*ast.Clause) {
		hasAccessors := func(c *ast.Clause) bool {
			return len(c.Pattern.(patternList).accessors) != 0
		}

		if utils.All(cs, hasAccessors) {
			// if all pattern lists have accessors, call Object recursively
			fields = append(fields, &ast.Field{Name: field, Exprs: []ast.Node{b.object(cs)}})
		} else if len(b.scrutinees) != 0 {
			// if any of cs has no accessors and has guards, generate Case expression

			/*
				case b.Scrutinee {
					caseClauses[0] (p0 -> e0)
					caseClauses[1] (p1 -> {restClauses})
					caseClauses[2] (p2 -> {restClauses})
				}
				(restClauses = caseClauses[1, 2])
			*/

			// case-clauses
			caseClauses := make([]*ast.Clause, len(cs))
			// case-clauses that have other accessors
			// for keeping order of clauses, use map[int]Clause instead of []Clause
			restClauses := make(map[int]*ast.Clause)
			for i, c := range cs {
				plist := c.Pattern.(patternList)
				if len(plist.accessors) == 0 {
					caseClauses[i] = &ast.Clause{
						Pattern: &ast.Paren{Elems: plist.params},
						Exprs:   c.Exprs,
					}
				} else {
					restClauses[i] = c
				}
			}

			restClausesList := make([]*ast.Clause, 0)
			utils.OrderedFor(restClauses, func(_ int, v *ast.Clause) {
				restClausesList = append(restClausesList, v)
			})

			for i, c := range restClauses {
				plist := c.Pattern.(patternList)
				caseClauses[i] = &ast.Clause{
					Pattern: &ast.Paren{Elems: plist.params},
					Exprs:   []ast.Node{b.object(restClausesList)},
				}
			}

			fields = append(fields,
				&ast.Field{
					Name: field,
					Exprs: []ast.Node{
						&ast.Case{
							Scrutinee: &ast.Paren{Elems: b.scrutinees},
							Clauses:   caseClauses,
						},
					},
				})
		} else {
			// if there is no scrutinee, simply insert the clause's body expression
			fields = append(fields,
				&ast.Field{
					Name:  field,
					Exprs: cs[0].Exprs,
				})
		}
	})
	return &ast.Object{Fields: fields}
}

// Generate lambda and dispatch body expression to Object or Case based on existence of accessors.
func (b *builder) lambda(arity int, clauses []*ast.Clause) ast.Node {
	baseToken := clauses[0].Base()
	// Generate Scrutinees
	b.scrutinees = make([]ast.Node, arity)
	for i := 0; i < arity; i++ {
		b.scrutinees[i] = &ast.Var{Name: token.Token{Kind: token.IDENT, Lexeme: fmt.Sprintf("x%d", i), Line: baseToken.Line, Literal: nil}}
	}

	// If any of clauses has accessors, body expression is Object.
	for _, c := range clauses {
		if len(c.Pattern.(patternList).accessors) != 0 {
			return &ast.Lambda{Pattern: &ast.Paren{Elems: b.scrutinees}, Exprs: []ast.Node{b.object(clauses)}}
		}
	}

	// otherwise, body expression is Case.
	caseClauses := make([]*ast.Clause, 0)
	for _, c := range clauses {
		plist := c.Pattern.(patternList)
		caseClauses = append(caseClauses, &ast.Clause{Pattern: &ast.Paren{Elems: plist.params}, Exprs: c.Exprs})
	}
	return &ast.Lambda{
		Pattern: &ast.Paren{Elems: b.scrutinees},
		Exprs: []ast.Node{
			&ast.Case{
				Scrutinee: &ast.Paren{Elems: b.scrutinees},
				Clauses:   caseClauses,
			},
		},
	}
}

func invalidPattern(n ast.Node) error {
	return utils.ErrorAt(n.Base(), fmt.Sprintf("invalid pattern %v", n))
}

// Collect all Access patterns recursively.
func accessors(p ast.Node) []token.Token {
	switch p := p.(type) {
	case *ast.Access:
		return append(accessors(p.Receiver), p.Name)
	default:
		return []token.Token{}
	}
}

// Get Args of Call{This, ...}
func params(p ast.Node) []ast.Node {
	switch p := p.(type) {
	case *ast.Access:
		return params(p.Receiver)
	case *ast.Call:
		if _, ok := p.Func.(*ast.This); !ok {
			panic(invalidPattern(p))
		}
		return p.Args
	default:
		return []ast.Node{}
	}
}

type patternList struct {
	accessors []token.Token
	params    []ast.Node
}

func (p patternList) Base() token.Token {
	if len(p.accessors) == 0 {
		if len(p.params) == 0 {
			return token.Token{}
		}
		return p.params[0].Base()
	}
	return p.accessors[0]
}

func (p patternList) String() string {
	var b strings.Builder
	b.WriteString("[")
	for i, a := range p.accessors {
		if i > 0 {
			b.WriteString(" ")
		}
		b.WriteString(a.String())
	}
	b.WriteString(" | ")
	for i, p := range p.params {
		if i > 0 {
			b.WriteString(" ")
		}
		b.WriteString(p.String())
	}
	b.WriteString("]")
	return b.String()
}

var _ ast.Node = patternList{}

// Split PatternList into the first accessor and the rest.
func pop(p patternList) (token.Token, patternList, bool) {
	if len(p.accessors) == 0 {
		return token.Token{}, p, false
	}
	a := p.accessors[0]
	p.accessors = p.accessors[1:]
	return a, p, true
}
