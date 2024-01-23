package rewrite

import (
	"fmt"
	"strings"

	"github.com/takoeight0821/anma/ast"
	"github.com/takoeight0821/anma/token"
)

type Observation struct {
	sequence []ast.Node // sequence is a sequence of patterns (destructors)
	current  int        // current is the index of the current pattern.
	body     []ast.Node
}

func (o Observation) String() string {
	var builder strings.Builder
	builder.WriteString("[ ")
	for _, s := range o.sequence {
		builder.WriteString(s.String())
		builder.WriteString(" ")
	}
	builder.WriteString("| ")
	for _, g := range o.Guard() {
		builder.WriteString(g.String())
		builder.WriteString(" ")
	}
	builder.WriteString("] -> { ")
	for _, e := range o.body {
		builder.WriteString(e.String())
		builder.WriteString("; ")
	}
	builder.WriteString("}")
	return builder.String()
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
func NewObservation(clause *ast.Clause) (*Observation, error) {
	if len(clause.Patterns) != 1 {
		return nil, NewInvalidPatternError(clause.Patterns...)
	}
	// Pattern must have a valid guard.
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
func extractGuard(pattern ast.Node) ([]ast.Node, error) {
	switch pattern := pattern.(type) {
	case *ast.Access:
		return extractGuard(pattern.Receiver)
	case *ast.Call:
		if _, ok := pattern.Func.(*ast.This); ok {
			return pattern.Args, nil
		}
	case *ast.This:
		return []ast.Node{}, nil
	}
	return nil, NewInvalidPatternError(pattern)
}

// extractSequence extracts sequence from the given pattern.
// Returns the sequence and error if the pattern is valid.
func extractSequence(pattern ast.Node) ([]ast.Node, error) {
	switch pattern := pattern.(type) {
	case *ast.Access:
		seq, err := extractSequence(pattern.Receiver)
		if err != nil {
			return nil, err
		}
		current := &ast.Access{Receiver: &ast.This{Token: pattern.Receiver.Base()}, Name: pattern.Name}
		return append(seq, current), nil
	case *ast.Call:
		seq, err := extractSequence(pattern.Func)
		if err != nil {
			return nil, err
		}
		current := &ast.Call{Func: &ast.This{Token: pattern.Func.Base()}, Args: pattern.Args}
		return append(seq, current), nil
	case *ast.This:
		return []ast.Node{}, nil
	default:
		return nil, NewInvalidPatternError(pattern)
	}
}

// Arity is the arity of the observation.
// For example: sequence, current -> Arity.
//
//	#(x, y) #.f #(z), 0 -> 2
//	#(x, y) #.f #(z), 1 -> -1
//	#(x, y) #.f #(z), 2 -> 1
//	#(x, y) #.f #(), 2 -> 0
type Arity int

const (
	// ArityNone is the arity of the observation with no traditional patterns.
	None Arity = -1
	// ArityZero is the arity of the observation with zero-arity function.
	Zero Arity = 0
)

// ArityOf returns the number of arguments of the observation.
// Panics if the observation includes invalid patterns.
func (o *Observation) ArityOf() Arity {
	switch o.Peek().(type) {
	case *ast.Access:
		return None
	case *ast.Call:
		callNode, ok := o.Peek().(*ast.Call)
		if !ok {
			panic("invalid pattern: not a call")
		}

		return Arity(len(callNode.Args))
	default:
		panic("invalid pattern")
	}
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
