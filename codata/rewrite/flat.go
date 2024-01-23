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

func (f *Flat) flat(node ast.Node) (ast.Node, error) {
	node, err := ast.Traverse(node, f.flatEach)
	if err != nil {
		return node, fmt.Errorf("flat %v: %w", node, err)
	}

	return node, nil
}

func (f *Flat) flatEach(node ast.Node, err error) (ast.Node, error) {
	// early return if error occurred.
	if err != nil {
		return node, err
	}
	if c, ok := node.(*ast.Codata); ok {
		flattened, err := f.flatCodata(c)
		if err != nil {
			return node, err
		}

		return flattened, nil
	}

	return node, nil
}

// flatCodata converts copatterns into object construction, function, and traditional pattern matching.
func (f *Flat) flatCodata(codata *ast.Codata) (ast.Node, error) {
	for _, clause := range codata.Clauses {
		ob, err := NewObservation(clause)
		if err != nil {
			return codata, err
		}
		log.Printf("observation of: %v => %v", clause.Patterns, ob.sequence)
	}

	return codata, nil
}
