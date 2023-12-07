package rewrite

import (
	"fmt"
	"log"
	"strings"

	"github.com/takoeight0821/anma/ast"
	"github.com/takoeight0821/anma/token"
)

// Flat converts copatterns into object construction, function, and traditional patterns.
type Flat struct {
	scrutinees []token.Token
}

func (Flat) Name() string {
	return "codata.flat"
}

func (Flat) Init([]ast.Node) error {
	return nil
}

func (f *Flat) Run(program []ast.Node) ([]ast.Node, error) {
	for _, n := range program {
		ast.Traverse(n, func(n ast.Node, _ error) (ast.Node, error) {
			if c, ok := n.(*ast.Clause); ok {
				ob := NewObservation(c)
				log.Printf("observation: %s", ob)
			}
			return n, nil
		})
	}
	return program, nil
}

type Observation struct {
	guard    []ast.Node // guard is patterns for branching.
	sequence []ast.Node // sequence is a sequence of patterns (destructors)
	body     []ast.Node
}

func (o Observation) String() string {
	var b strings.Builder
	b.WriteString("[ ")
	for _, s := range o.sequence {
		b.WriteString(s.String())
		b.WriteString(" ")
	}
	b.WriteString("| ")
	for _, g := range o.guard {
		b.WriteString(g.String())
		b.WriteString(" ")
	}
	b.WriteString("] -> { ")
	for _, e := range o.body {
		b.WriteString(e.String())
		b.WriteString("; ")
	}
	b.WriteString("}")
	return b.String()
}

func (o Observation) Base() token.Token {
	if len(o.sequence) != 0 {
		return o.sequence[0].Base()
	}
	if len(o.guard) != 0 {
		return o.guard[0].Base()
	}
	return token.Token{}
}

func (o *Observation) Plate(err error, f func(ast.Node, error) (ast.Node, error)) (ast.Node, error) {
	for i, s := range o.sequence {
		o.sequence[i], err = f(s, err)
	}
	for i, g := range o.guard {
		o.guard[i], err = f(g, err)
	}
	return o, err
}

var _ ast.Node = &Observation{}

// NewObservation creates a new observation node with the given pattern.
func NewObservation(clause *ast.Clause) *Observation {
	return &Observation{
		guard:    extractGuard(clause.Patterns[0]),
		sequence: extractSequence(clause.Patterns[0]),
		body:     clause.Exprs,
	}
}

// extractGuard extracts guard from the given pattern.
func extractGuard(p ast.Node) []ast.Node {
	switch p := p.(type) {
	case *ast.Access:
		return extractGuard(p.Receiver)
	case *ast.Call:
		if _, ok := p.Func.(*ast.This); ok {
			return p.Args
		}
	case *ast.This:
		return []ast.Node{}
	}
	panic(fmt.Sprintf("invalid pattern %v", p))
}

// extractSequence extracts sequence from the given pattern.
func extractSequence(p ast.Node) []ast.Node {
	switch p := p.(type) {
	case *ast.Access:
		current := &ast.Access{Receiver: &ast.This{Token: p.Receiver.Base()}, Name: p.Name}
		return append(extractSequence(p.Receiver), current)
	case *ast.Call:
		if _, ok := p.Func.(*ast.This); !ok {
			panic(fmt.Sprintf("invalid pattern %v", p))
		}
		return []ast.Node{p}
	case *ast.This:
		return []ast.Node{}
	default:
		panic(fmt.Sprintf("invalid pattern %v", p))
	}
}

// ArityOf returns the number of arguments of the observation.
func (o *Observation) ArityOf() int {
	return len(o.guard)
}

func (o *Observation) Pop() (ast.Node, *Observation, bool) {
	if len(o.sequence) == 0 {
		return nil, nil, false
	}
	return o.sequence[0], &Observation{guard: o.guard, sequence: o.sequence[1:]}, true
}

func (o *Observation) HasAccess() bool {
	if len(o.sequence) == 0 {
		return false
	}

	switch o.sequence[0].(type) {
	case *ast.Access:
		return true
	default:
		return false
	}
}

func (o *Observation) HasCall() bool {
	if len(o.sequence) == 0 {
		return false
	}

	switch o.sequence[0].(type) {
	case *ast.Call:
		return true
	default:
		return false
	}
}

func (o *Observation) Guard() []ast.Node {
	return o.guard
}

func (o *Observation) toClause() *ast.Clause {
	return &ast.Clause{
		Patterns: o.Guard(),
		Exprs:    o.body,
	}
}
