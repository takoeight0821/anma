package main

import (
	"fmt"
	"strings"
)

// [Flat] converts Copatterns ([Access] and [This] in [Pattern]) into [Object] and [Lambda].
func Flat(n Node) Node {
	return Transform(n, flat)
}

func flat(n Node) Node {
	if n, ok := n.(Codata); ok {
		return flatCodata(n)
	}
	return n
}

func flatCodata(c Codata) Node {
	// Generate PatternList
	arity := -1
	for i, cl := range c.Clauses {
		plist := PatternList{Accessors: accessors(cl.Pattern), Params: params(cl.Pattern)}
		c.Clauses[i] = Clause{Pattern: plist, Exprs: cl.Exprs}
		if arity == -1 {
			arity = len(plist.Params)
		} else if arity != len(plist.Params) {
			panic(fmt.Errorf("arity mismatch at %d: %v", c.Base().Line, c))
		}
	}

	if arity == -1 {
		panic(fmt.Errorf("unreachable: arity is -1 at %d: %v", c.Base().Line, c))
	}

	return NewBuilder().Build(arity, c.Clauses)
}

type Builder struct {
	Scrutinees []Node
}

func NewBuilder() *Builder {
	return &Builder{}
}

// dispatch to Object or Lambda based on arity
func (b *Builder) Build(arity int, clauses []Clause) Node {
	if arity == 0 {
		return b.Object(clauses)
	}
	return b.Lambda(arity, clauses)
}

func (b *Builder) Object(clauses []Clause) Object {
	// Pop the first accessor of each clause and group remaining clauses by the popped accessor.
	next := make(map[string][]Clause)
	for _, c := range clauses {
		plist := c.Pattern.(PatternList)
		if field, plist, ok := Pop(plist); ok {
			next[field.String()] = append(
				next[field.String()],
				Clause{Pattern: plist, Exprs: c.Exprs})
		} else {
			panic(fmt.Errorf("not implemented: %v\nmix of pure pattern and copattern is not supported yet", c))
		}
	}

	fields := make([]Field, 0)

	// Generate each field's body expression
	// Object fields are generated in the dictionary order of field names.
	orderedFor(next, func(field string, cs []Clause) {
		hasAccessors := func(c Clause) bool {
			return len(c.Pattern.(PatternList).Accessors) != 0
		}

		if all(cs, hasAccessors) {
			// if all pattern lists have accessors, call Object recursively
			fields = append(fields, Field{Name: field, Exprs: []Node{b.Object(cs)}})
		} else if len(b.Scrutinees) != 0 {
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
			caseClauses := make([]Clause, len(cs))
			// case-clauses that have other accessors
			// for keeping order of clauses, use map[int]Clause instead of []Clause
			restClauses := make(map[int]Clause)
			for i, c := range cs {
				plist := c.Pattern.(PatternList)
				if len(plist.Accessors) == 0 {
					caseClauses[i] = Clause{
						Pattern: Paren{Elems: plist.Params},
						Exprs:   c.Exprs,
					}
				} else {
					restClauses[i] = c
				}
			}

			restClausesList := make([]Clause, 0)
			orderedFor(restClauses, func(_ int, v Clause) {
				restClausesList = append(restClausesList, v)
			})

			for i, c := range restClauses {
				plist := c.Pattern.(PatternList)
				caseClauses[i] = Clause{
					Pattern: Paren{Elems: plist.Params},
					Exprs:   []Node{b.Object(restClausesList)},
				}
			}

			fields = append(fields,
				Field{
					Name: field,
					Exprs: []Node{
						Case{
							Scrutinee: Paren{Elems: b.Scrutinees},
							Clauses:   caseClauses,
						},
					},
				})
		} else {
			// if there is no scrutinee, simply insert the clause's body expression
			fields = append(fields,
				Field{
					Name:  field,
					Exprs: cs[0].Exprs,
				})
		}
	})
	return Object{Fields: fields}
}

// Generate Lambda and dispatch body expression to Object or Case based on existence of accessors.
func (b *Builder) Lambda(arity int, clauses []Clause) Lambda {
	baseToken := Codata{Clauses: clauses}.Base()
	// Generate Scrutinees
	b.Scrutinees = make([]Node, arity)
	for i := 0; i < arity; i++ {
		b.Scrutinees[i] = Var{Name: Token{Kind: IDENT, Lexeme: fmt.Sprintf("x%d", i), Line: baseToken.Line, Literal: nil}}
	}

	// If any of clauses has accessors, body expression is Object.
	for _, c := range clauses {
		if len(c.Pattern.(PatternList).Accessors) != 0 {
			return Lambda{Pattern: Paren{Elems: b.Scrutinees}, Exprs: []Node{b.Object(clauses)}}
		}
	}

	// otherwise, body expression is Case.
	caseClauses := make([]Clause, 0)
	for _, c := range clauses {
		plist := c.Pattern.(PatternList)
		caseClauses = append(caseClauses, Clause{Pattern: Paren{Elems: plist.Params}, Exprs: c.Exprs})
	}
	return Lambda{
		Pattern: Paren{Elems: b.Scrutinees},
		Exprs: []Node{
			Case{
				Scrutinee: Paren{Elems: b.Scrutinees},
				Clauses:   caseClauses,
			},
		},
	}
}

func InvalidPattern(n Node) error {
	return fmt.Errorf("invalid pattern: %v", n)
}

// Collect all Access patterns recursively.
func accessors(p Node) []Token {
	switch p := p.(type) {
	case Access:
		return append(accessors(p.Receiver), p.Name)
	default:
		return []Token{}
	}
}

// Get Args of Call{This, ...}
func params(p Node) []Node {
	switch p := p.(type) {
	case Access:
		return params(p.Receiver)
	case Call:
		if _, ok := p.Func.(This); !ok {
			panic(InvalidPattern(p))
		}
		return p.Args
	default:
		return []Node{}
	}
}

type PatternList struct {
	Accessors []Token
	Params    []Node
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

var _ Node = PatternList{}

// Split PatternList into the first accessor and the rest.
func Pop(p PatternList) (Token, PatternList, bool) {
	if len(p.Accessors) == 0 {
		return Token{}, p, false
	}
	a := p.Accessors[0]
	p.Accessors = p.Accessors[1:]
	return a, p, true
}
