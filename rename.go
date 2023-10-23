package main

import "fmt"

type Renamer struct {
	supply int
	env    *Env
}

func NewRenamer() *Renamer {
	return &Renamer{supply: 0, env: NewEnv(nil)}
}

func (r *Renamer) scoped(f func()) {
	r.env = NewEnv(r.env)
	f()
	r.env = r.env.parent
}

func (r *Renamer) assign(node Node) {
	Transform(node, func(n Node) Node {
		switch n := n.(type) {
		case *Var:
			r.new(n.Name.Lexeme)
		}
		return n
	})
}

func (r *Renamer) new(name string) int {
	if _, ok := r.env.table[name]; ok {
		panic(fmt.Errorf("%v is already defined", name))
	}
	r.env.table[name] = r.unique()
	return r.env.table[name]
}

func (r *Renamer) unique() int {
	u := r.supply
	r.supply++
	return u
}

func (r *Renamer) lookup(name string) int {
	if uniq, err := r.env.lookup(name); err != nil {
		panic(err)
	} else {
		return uniq
	}
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
			r.assign(n.Bind)
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
			r.assign(n.Pattern)
			n.Pattern = r.Solve(n.Pattern)
			for i, expr := range n.Exprs {
				n.Exprs[i] = r.Solve(expr)
			}
		})
		return n
	case *Lambda:
		r.scoped(func() {
			r.assign(n.Pattern)
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
		r.scoped(func() {
			for i, field := range n.Fields {
				n.Fields[i] = r.Solve(field).(*Field)
			}
		})
		return n
	case *TypeDecl:
		n.Name.Literal = r.new(n.Name.Lexeme)
		n.Type = r.Solve(n.Type)
		return n
	case *VarDecl:
		n.Name.Literal = r.new(n.Name.Lexeme)
		if n.Type != nil {
			n.Type = r.Solve(n.Type)
		}
		if n.Expr != nil {
			n.Expr = r.Solve(n.Expr)
		}
		return n
	case *InfixDecl:
		return n
	case *This:
		return n
	default:
		panic(fmt.Errorf("Renamer.Solve not implemented: %v", n))
	}
}
