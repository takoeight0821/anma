package main

import (
	"errors"
	"fmt"
)

type Renamer struct {
	supply int
	env    *Env
	err    error
}

func NewRenamer() *Renamer {
	return &Renamer{supply: 0, env: NewEnv(nil), err: nil}
}

func (r *Renamer) PopError() error {
	err := r.err
	r.err = nil
	return err
}

func (r *Renamer) error(err error) {
	r.err = errors.Join(r.err, err)
}

func (r *Renamer) scoped(nodes []Node, f func()) {
	r.env = NewEnv(r.env)
	for _, node := range nodes {
		r.assign(node)
	}
	f()
	if r.err != nil {
		for _, node := range nodes {
			r.delete(node)
		}
	}
	r.env = r.env.parent
}

func (r *Renamer) assign(node Node) {
	addTable := func(name string) {
		if _, ok := r.env.table[name]; ok {
			r.error(fmt.Errorf("%v is already defined", name))
			return
		}
		r.env.table[name] = r.unique()
	}
	Transform(node, func(n Node) Node {
		switch n := n.(type) {
		case *Var:
			addTable(n.Name.Lexeme)
		case Token:
			addTable(n.Lexeme)
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
		case Token:
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

func (r *Renamer) lookup(name string) int {
	uniq, err := r.env.lookup(name)
	if err != nil {
		r.error(err)
	}
	return uniq
}

type Env struct {
	table  map[string]int
	parent *Env
}

func NewEnv(parent *Env) *Env {
	return &Env{table: make(map[string]int), parent: parent}
}

func (e *Env) lookup(name string) (int, error) {
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
		n.Name.Literal = r.lookup(n.Name.Lexeme)
		return n
	case *Literal:
		return n
	case *Paren:
		r.scoped(nil, func() {
			for i, elem := range n.Elems {
				n.Elems[i] = r.Solve(elem)
			}
		})
		return n
	case *Access:
		r.scoped(nil, func() {
			n.Receiver = r.Solve(n.Receiver)
		})
		return n
	case *Call:
		r.scoped(nil, func() {
			n.Func = r.Solve(n.Func)
			for _, arg := range n.Args {
				r.Solve(arg)
			}
		})
		return n
	case *Binary:
		r.scoped(nil, func() {
			n.Left = r.Solve(n.Left)
			n.Right = r.Solve(n.Right)
		})
		return n
	case *Assert:
		r.scoped(nil, func() {
			n.Expr = r.Solve(n.Expr)
			n.Type = r.Solve(n.Type)
		})
		return n
	case *Let:
		r.scoped([]Node{n.Bind}, func() {
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
		r.scoped([]Node{n.Pattern}, func() {
			n.Pattern = r.Solve(n.Pattern)
			for i, expr := range n.Exprs {
				n.Exprs[i] = r.Solve(expr)
			}
		})
		return n
	case *Lambda:
		r.scoped([]Node{n.Pattern}, func() {
			n.Pattern = r.Solve(n.Pattern)
			for i, expr := range n.Exprs {
				n.Exprs[i] = r.Solve(expr)
			}
		})
		return n
	case *Case:
		r.scoped(nil, func() {
			n.Scrutinee = r.Solve(n.Scrutinee)
			for i, clause := range n.Clauses {
				n.Clauses[i] = r.Solve(clause).(*Clause)
			}
		})
		return n
	case *Object:
		r.scoped(nil, func() {
			for i, field := range n.Fields {
				n.Fields[i] = r.Solve(field).(*Field)
			}
		})
		return n
	case *TypeDecl:
		r.assign(n.Name)
		n.Name.Literal = r.lookup(n.Name.Lexeme)
		n.Type = r.Solve(n.Type)
		if r.err != nil {
			r.delete(n.Name)
		}
		return n
	case *VarDecl:
		r.assign(n.Name)
		n.Name.Literal = r.lookup(n.Name.Lexeme)
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
		r.error(fmt.Errorf("Renamer.Solve not implemented: %v", n))
		return n
	}
}
