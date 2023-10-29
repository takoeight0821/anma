package main

import (
	"errors"
	"fmt"

	"github.com/takoeight0821/anma/internal/token"
)

type Renamer struct {
	supply int
	env    *rnEnv
	err    error
}

func NewRenamer() *Renamer {
	return &Renamer{supply: 0, env: newRnEnv(nil), err: nil}
}

func (r *Renamer) Init(program []Node) error {
	return nil
}

func (r *Renamer) Run(program []Node) ([]Node, error) {
	for i, n := range program {
		program[i] = r.Solve(n)
	}

	return program, r.popError()
}

func (r *Renamer) popError() error {
	err := r.err
	r.err = nil
	return err
}

func (r *Renamer) error(err error) {
	r.err = errors.Join(r.err, err)
}

func (r *Renamer) scoped(f func()) {
	r.env = newRnEnv(r.env)
	f()
	r.env = r.env.parent
}

func (r *Renamer) assign(node Node, overridable bool) {
	addTable := func(name token.Token) {
		if _, ok := r.env.table[name.Lexeme]; ok && !overridable {
			r.error(errorAt(name.Base(), fmt.Sprintf("%v is already defined", name)))
			return
		}
		r.env.table[name.Lexeme] = r.unique()
	}
	Transform(node, func(n Node) Node {
		switch n := n.(type) {
		case *Var:
			addTable(n.Name)
		case token.Token:
			addTable(n)
		}
		return n
	})
}

func (r *Renamer) delete(node Node) {
	deleteTable := func(name string) {
		delete(r.env.table, name)
	}
	Transform(node, func(n Node) Node {
		switch n := n.(type) {
		case *Var:
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

func (r *Renamer) lookup(name token.Token) int {
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

func newRnEnv(parent *rnEnv) *rnEnv {
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

func (r *Renamer) Solve(node Node) Node {
	switch n := node.(type) {
	case *Var:
		n.Name.Literal = r.lookup(n.Name)
		return n
	case *Literal:
		return n
	case *Paren:
		r.scoped(func() {
			for i, elem := range n.Elems {
				n.Elems[i] = r.Solve(elem)
			}
		})
		return n
	case *Access:
		r.scoped(func() {
			n.Receiver = r.Solve(n.Receiver)
		})
		return n
	case *Call:
		r.scoped(func() {
			n.Func = r.Solve(n.Func)
			for _, arg := range n.Args {
				r.Solve(arg)
			}
		})
		return n
	case *Binary:
		r.scoped(func() {
			n.Left = r.Solve(n.Left)
			n.Right = r.Solve(n.Right)
		})
		return n
	case *Assert:
		r.scoped(func() {
			n.Expr = r.Solve(n.Expr)
			n.Type = r.Solve(n.Type)
		})
		return n
	case *Let:
		r.scoped(func() {
			r.assign(n.Bind, false)
			n.Bind = r.Solve(n.Bind)
			n.Body = r.Solve(n.Body)
		})
		return n
	case *Codata:
		for i, clause := range n.Clauses {
			n.Clauses[i] = r.Solve(clause).(*Clause)
		}
		return n
	case *Clause:
		r.scoped(func() {
			r.assign(n.Pattern, false)
			n.Pattern = r.Solve(n.Pattern)
			for i, expr := range n.Exprs {
				n.Exprs[i] = r.Solve(expr)
			}
		})
		return n
	case *Lambda:
		r.scoped(func() {
			r.assign(n.Pattern, false)
			n.Pattern = r.Solve(n.Pattern)
			for i, expr := range n.Exprs {
				n.Exprs[i] = r.Solve(expr)
			}
		})
		return n
	case *Case:
		r.scoped(func() {
			n.Scrutinee = r.Solve(n.Scrutinee)
			for i, clause := range n.Clauses {
				n.Clauses[i] = r.Solve(clause).(*Clause)
			}
		})
		return n
	case *Object:
		for i, field := range n.Fields {
			n.Fields[i] = r.Solve(field).(*Field)
		}
		return n
	case *Field:
		r.scoped(func() {
			for i, expr := range n.Exprs {
				n.Exprs[i] = r.Solve(expr)
			}
		})
		return n
	case *TypeDecl:
		// Type definition can override existential definition
		r.assign(n.Name, true)
		n.Name.Literal = r.lookup(n.Name)
		n.Type = r.Solve(n.Type)
		if r.err != nil {
			r.delete(n.Name)
		}
		return n
	case *VarDecl:
		// Toplevel variable definition can override existential definition
		r.assign(n.Name, true)
		n.Name.Literal = r.lookup(n.Name)
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
	case *InfixDecl:
		return n
	case *This:
		return n
	default:
		r.error(errorAt(n.Base(), fmt.Sprintf("Renamer.Solve not implemented: %v", n)))
		return n
	}
}
