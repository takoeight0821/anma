package rewrite

import (
	"fmt"
	"log"

	"github.com/takoeight0821/anma/ast"
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

// flatCodata converts copatterns into object construction, function, and traditional pattern matching.
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
