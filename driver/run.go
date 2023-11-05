package driver

import (
	"fmt"

	"github.com/takoeight0821/anma/ast"
	"github.com/takoeight0821/anma/lexer"
	"github.com/takoeight0821/anma/parser"
)

type Pass interface {
	Init([]ast.Node) error
	Run([]ast.Node) ([]ast.Node, error)
}

type PassRunner struct {
	passes []Pass
}

func NewPassRunner() *PassRunner {
	return &PassRunner{}
}

// AddPass adds a pass to the end of the pass list.
func (r *PassRunner) AddPass(pass Pass) {
	r.passes = append(r.passes, pass)
}

// Run executes passes in order.
// If an error occurs, it stops the execution and returns the current program.
func (r *PassRunner) Run(program []ast.Node) ([]ast.Node, error) {
	for _, pass := range r.passes {
		err := pass.Init(program)
		if err != nil {
			return program, fmt.Errorf("init: %w", err)
		}
		program, err = pass.Run(program)
		if err != nil {
			return program, fmt.Errorf("run: %w", err)
		}
	}

	return program, nil
}

// RunSource parses the source code and executes passes in order.
func (r *PassRunner) RunSource(source string) ([]ast.Node, error) {
	tokens, err := lexer.Lex(source)
	if err != nil {
		return nil, fmt.Errorf("lex: %w", err)
	}

	var program []ast.Node
	if decls, err := parser.NewParser(tokens).ParseDecl(); err == nil {
		program = decls
	} else if expr, err := parser.NewParser(tokens).ParseExpr(); err == nil {
		program = []ast.Node{expr}
	} else {
		return nil, fmt.Errorf("parse: %w", err)
	}

	return r.Run(program)
}
