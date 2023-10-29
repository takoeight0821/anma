// Package rename provides a renamer that assigns unique integer to each variable.
package rename

import (
	"errors"
	"fmt"

	"github.com/takoeight0821/anma/ast"
	"github.com/takoeight0821/anma/token"
	"github.com/takoeight0821/anma/utils"
)

// Renamer assigns unique integer to each variable.
type Renamer struct {
	supply int
	env    *rnEnv
	err    error
}

func NewRenamer() *Renamer {
	return &Renamer{supply: 0, env: NewRnEnv(nil), err: nil}
}

func (r *Renamer) Init(program []ast.Node) error {
	return nil
}

func (r *Renamer) Run(program []ast.Node) ([]ast.Node, error) {
	for i, n := range program {
		program[i] = r.Solve(n)
	}

	return program, r.PopError()
}

// PopError returns the error that occurred during the last Run.
// If several errors occurred, PopError returns concatenated error.
// And then, PopError resets the error.
func (r *Renamer) PopError() error {
	err := r.err
	r.err = nil
	return err
}

func (r *Renamer) error(err error) {
	r.err = errors.Join(r.err, err)
}

func (r *Renamer) scoped(f func()) {
	r.env = NewRnEnv(r.env)
	f()
	r.env = r.env.parent
}

func (r *Renamer) assign(node ast.Node, overridable bool) {
	addTable := func(name token.Token) {
		if _, ok := r.env.table[name.Lexeme]; ok && !overridable {
			r.error(utils.ErrorAt(name.Base(), fmt.Sprintf("%v is already defined", name)))
			return
		}
		r.env.table[name.Lexeme] = r.unique()
	}
	ast.Transform(node, func(n ast.Node) ast.Node {
		switch n := n.(type) {
		case *ast.Var:
			addTable(n.Name)
		case token.Token:
			addTable(n)
		}
		return n
	})
}

func (r *Renamer) delete(node ast.Node) {
	deleteTable := func(name string) {
		delete(r.env.table, name)
	}
	ast.Transform(node, func(n ast.Node) ast.Node {
		switch n := n.(type) {
		case *ast.Var:
			deleteTable(n.Name.Lexeme)
		case token.Token:
			deleteTable(n.Lexeme)
		}
		return n
	})
}

func (r *Renamer) unique() int {
	u := r.supply
	r.supply++
	return u
}

func (r *Renamer) Lookup(name token.Token) int {
	uniq, err := r.env.lookup(name.Lexeme)
	if err != nil {
		r.error(err)
	}
	return uniq
}

type rnEnv struct {
	table  map[string]int
	parent *rnEnv
}

func NewRnEnv(parent *rnEnv) *rnEnv {
	return &rnEnv{table: make(map[string]int), parent: parent}
}

func (e *rnEnv) lookup(name string) (int, error) {
	if uniq, ok := e.table[name]; ok {
		return uniq, nil
	}
	if e.parent != nil {
		return e.parent.lookup(name)
	}
	return -1, fmt.Errorf("%v is not defined", name)
}

func (r *Renamer) Solve(node ast.Node) ast.Node {
	switch n := node.(type) {
	case *ast.Var:
		n.Name.Literal = r.Lookup(n.Name)
		return n
	case *ast.Literal:
		return n
	case *ast.Paren:
		r.scoped(func() {
			for i, elem := range n.Elems {
				n.Elems[i] = r.Solve(elem)
			}
		})
		return n
	case *ast.Access:
		r.scoped(func() {
			n.Receiver = r.Solve(n.Receiver)
		})
		return n
	case *ast.Call:
		r.scoped(func() {
			n.Func = r.Solve(n.Func)
			for _, arg := range n.Args {
				r.Solve(arg)
			}
		})
		return n
	case *ast.Binary:
		r.scoped(func() {
			n.Left = r.Solve(n.Left)
			n.Right = r.Solve(n.Right)
		})
		return n
	case *ast.Assert:
		r.scoped(func() {
			n.Expr = r.Solve(n.Expr)
			n.Type = r.Solve(n.Type)
		})
		return n
	case *ast.Let:
		r.scoped(func() {
			r.assign(n.Bind, false)
			n.Bind = r.Solve(n.Bind)
			n.Body = r.Solve(n.Body)
		})
		return n
	case *ast.Codata:
		for i, clause := range n.Clauses {
			n.Clauses[i] = r.Solve(clause).(*ast.Clause)
		}
		return n
	case *ast.Clause:
		r.scoped(func() {
			r.assign(n.Pattern, false)
			n.Pattern = r.Solve(n.Pattern)
			for i, expr := range n.Exprs {
				n.Exprs[i] = r.Solve(expr)
			}
		})
		return n
	case *ast.Lambda:
		r.scoped(func() {
			r.assign(n.Pattern, false)
			n.Pattern = r.Solve(n.Pattern)
			for i, expr := range n.Exprs {
				n.Exprs[i] = r.Solve(expr)
			}
		})
		return n
	case *ast.Case:
		r.scoped(func() {
			n.Scrutinee = r.Solve(n.Scrutinee)
			for i, clause := range n.Clauses {
				n.Clauses[i] = r.Solve(clause).(*ast.Clause)
			}
		})
		return n
	case *ast.Object:
		for i, field := range n.Fields {
			n.Fields[i] = r.Solve(field).(*ast.Field)
		}
		return n
	case *ast.Field:
		r.scoped(func() {
			for i, expr := range n.Exprs {
				n.Exprs[i] = r.Solve(expr)
			}
		})
		return n
	case *ast.TypeDecl:
		// Type definition can override existential definition
		r.assign(n.Name, true)
		n.Name.Literal = r.Lookup(n.Name)
		n.Type = r.Solve(n.Type)
		if r.err != nil {
			r.delete(n.Name)
		}
		return n
	case *ast.VarDecl:
		// Toplevel variable definition can override existential definition
		r.assign(n.Name, true)
		n.Name.Literal = r.Lookup(n.Name)
		if n.Type != nil {
			n.Type = r.Solve(n.Type)
		}
		if n.Expr != nil {
			n.Expr = r.Solve(n.Expr)
		}
		if r.err != nil {
			r.delete(n.Name)
		}
		return n
	case *ast.InfixDecl:
		return n
	case *ast.This:
		return n
	default:
		r.error(utils.ErrorAt(n.Base(), fmt.Sprintf("Renamer.Solve not implemented: %v", n)))
		return n
	}
}
