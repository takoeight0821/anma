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
	for i, n := range program {
		var err error
		program[i], err = f.flat(n)
		if err != nil {
			return program, err
		}
	}
	return program, nil
}

func (f *Flat) flat(n ast.Node) (ast.Node, error) {
	n, err := ast.Traverse(n, f.flatEach)
	if err != nil {
		return n, fmt.Errorf("flat %v: %w", n, err)
	}
	return n, nil
}

func (f *Flat) flatEach(n ast.Node, err error) (ast.Node, error) {
	// early return if error occurred.
	if err != nil {
		return n, err
	}
	if c, ok := n.(*ast.Codata); ok {
		n2, err := f.flatCodata(c)
		if err != nil {
			return n, err
		}
		return n2, nil
	}
	return n, nil
}

func (f *Flat) flatCodata(c *ast.Codata) (ast.Node, error) {
	for _, clause := range c.Clauses {
		ob := NewObservation(clause)
		log.Printf("observation of: %v => %v", clause.Patterns, ob.sequence)
	}
	return c, nil
}

type Observation struct {
	sequence []ast.Node // sequence is a sequence of patterns (destructors)
	current  int        // current is the index of the current pattern.
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
	for _, g := range o.Guard() {
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
	return token.Token{}
}

func (o *Observation) Plate(err error, f func(ast.Node, error) (ast.Node, error)) (ast.Node, error) {
	for i, s := range o.sequence {
		o.sequence[i], err = f(s, err)
	}
	return o, err
}

var _ ast.Node = &Observation{}

// NewObservation creates a new observation node with the given pattern.
func NewObservation(clause *ast.Clause) *Observation {
	return &Observation{
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
		current := &ast.Call{Func: &ast.This{Token: p.Func.Base()}, Args: p.Args}
		return append(extractSequence(p.Func), current)
	case *ast.This:
		return []ast.Node{}
	default:
		panic(fmt.Sprintf("invalid pattern %v", p))
	}
}

// ArityOf returns the number of arguments of the observation.
func (o *Observation) ArityOf() int {
	return len(o.Guard())
}

func (o *Observation) Guard() []ast.Node {
	return extractGuard(o.sequence[0])
}

func (o *Observation) IsEmpty() bool {
	return len(o.sequence)-o.current <= 0
}

func (o *Observation) Peek() ast.Node {
	if o.IsEmpty() {
		return nil
	}
	return o.sequence[o.current]
}

func (o *Observation) Pop() (ast.Node, *Observation, bool) {
	if o.IsEmpty() {
		return o.Peek(), nil, false
	}
	return o.Peek(), &Observation{sequence: o.sequence, current: o.current + 1, body: o.body}, true
}

func (o *Observation) HasAccess() bool {
	if o.IsEmpty() {
		return false
	}

	switch o.Peek().(type) {
	case *ast.Access:
		return true
	default:
		return false
	}
}

func (o *Observation) HasCall() bool {
	if o.IsEmpty() {
		return false
	}

	switch o.Peek().(type) {
	case *ast.Call:
		return true
	default:
		return false
	}
}

func (o *Observation) toClause() *ast.Clause {
	return &ast.Clause{
		Patterns: o.Guard(),
		Exprs:    o.body,
	}
}
