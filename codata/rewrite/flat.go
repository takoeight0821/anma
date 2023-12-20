package rewrite

import (
	"fmt"
	"log"
	"strings"

	"github.com/takoeight0821/anma/ast"
	"github.com/takoeight0821/anma/token"
)

// Flat converts copatterns into object construction, function, and traditional patterns.
type Flat struct{}

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
		ob, err := NewObservation(clause)
		if err != nil {
			return c, err
		}
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

type InvalidPatternError struct {
	Patterns []ast.Node
}

func (e *InvalidPatternError) Error() string {
	return fmt.Sprintf("invalid pattern: %v", e.Patterns)
}

func NewInvalidPatternError(patterns ...ast.Node) *InvalidPatternError {
	return &InvalidPatternError{Patterns: patterns}
}

// NewObservation creates a new observation node with the given pattern.
func NewObservation(clause *ast.Clause) (*Observation, error) {
	if len(clause.Patterns) != 1 {
		return nil, NewInvalidPatternError(clause.Patterns...)
	}
	_, err := extractGuard(clause.Patterns[0])
	if err != nil {
		return nil, err
	}
	seq, err := extractSequence(clause.Patterns[0])
	if err != nil {
		return nil, err
	}

	return &Observation{
		sequence: seq,
		body:     clause.Exprs,
	}, nil
}

// extractGuard extracts guard from the given pattern.
// Returns the guard and error if the pattern is valid.
func extractGuard(p ast.Node) ([]ast.Node, error) {
	switch p := p.(type) {
	case *ast.Access:
		return extractGuard(p.Receiver)
	case *ast.Call:
		if _, ok := p.Func.(*ast.This); ok {
			return p.Args, nil
		}
	case *ast.This:
		return []ast.Node{}, nil
	}
	return nil, NewInvalidPatternError(p)
}

// extractSequence extracts sequence from the given pattern.
// Returns the sequence and error if the pattern is valid.
func extractSequence(p ast.Node) ([]ast.Node, error) {
	switch p := p.(type) {
	case *ast.Access:
		seq, err := extractSequence(p.Receiver)
		if err != nil {
			return nil, err
		}
		current := &ast.Access{Receiver: &ast.This{Token: p.Receiver.Base()}, Name: p.Name}
		return append(seq, current), nil
	case *ast.Call:
		seq, err := extractSequence(p.Func)
		if err != nil {
			return nil, err
		}
		current := &ast.Call{Func: &ast.This{Token: p.Func.Base()}, Args: p.Args}
		return append(seq, current), nil
	case *ast.This:
		return []ast.Node{}, nil
	default:
		return nil, NewInvalidPatternError(p)
	}
}

// ArityOf returns the number of arguments of the observation.
// Panics if the observation includes invalid patterns.
func (o *Observation) ArityOf() int {
	return len(o.Guard())
}

// Scrutinees returns the Scrutinees of the observation.
// For example: sequence, current -> Scrutinees.
//
//	#(x, y) #.f #(z), 0 -> [0, 1]
//	#(x, y) #.f #(z), 2 -> [0, 1, 2]
func (o *Observation) Scrutinees() []token.Token {
	scrutinees := []token.Token{}
	for i, s := range o.sequence {
		if i <= o.current {
			scrName := token.Token{Kind: token.IDENT, Lexeme: fmt.Sprintf("%%scr%d", i), Line: s.Base().Line, Literal: nil}
			scrutinees = append(scrutinees, scrName)
		}
	}

	return scrutinees
}

// Guard returns the guard of the observation.
// For example: sequence, current -> Guard.
//
//	#(x, y) #.f #(z), 0 -> [x, y]
//	#(x, y) #.f #(z), 2 -> [x, y, z]
//
// Panics if the observation includes invalid patterns.
func (o *Observation) Guard() []ast.Node {
	guards := []ast.Node{}
	for i, s := range o.sequence {
		if i <= o.current {
			guard, err := extractGuard(s)
			if err != nil {
				panic(err)
			}
			guards = append(guards, guard...)
		}
	}

	return guards
}

// IsEmpty returns true if the observation is empty.
func (o *Observation) IsEmpty() bool {
	return len(o.sequence)-o.current <= 0
}

// Peek returns the current pattern of the observation.
// If the observation is empty, it returns nil.
func (o *Observation) Peek() ast.Node {
	if o.IsEmpty() {
		return nil
	}
	return o.sequence[o.current]
}

// Pop returns the current pattern of the observation and the rest of the observation.
// The rest of the observation is newly allocated and shares the same sequence and body.
// If the observation is empty, it returns nil and nil.
func (o *Observation) Pop() (ast.Node, *Observation) {
	if o.IsEmpty() {
		return o.Peek(), nil
	}
	return o.Peek(), &Observation{sequence: o.sequence, current: o.current + 1, body: o.body}
}

// IsAccess returns true if the current pattern is access.
func (o *Observation) IsAccess() bool {
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

// IsCall returns true if the current pattern is call.
func (o *Observation) IsCall() bool {
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

// toClause converts the observation to a clause.
// Panics if the observation includes invalid patterns.
func (o *Observation) toClause() *ast.Clause {
	return &ast.Clause{
		Patterns: o.Guard(),
		Exprs:    o.body,
	}
}
