package ast

import (
	"log"

	"github.com/takoeight0821/anma/token"
)

type Repr[T any] interface {
	Var(name token.Token) T
	Literal(value token.Token) T
	Paren(expr T) T
	Access(receiver T, name token.Token) T
	Call(callee T, args []T) T
	Prim(name token.Token, args []T) T
	Binary(left T, op token.Token, right T) T
	Assert(expr T, typ T) T
	Let(bind T, body T) T
	Codata(clauses []T) T
	Clause(patterns []T, exprs []T) T
	Lambda(params []token.Token, exprs []T) T
	Case(scrutinees []T, clauses []T) T
	Object(fields []T) T
	Field(name string, exprs []T) T
	TypeDecl(def T, types []T) T
	VarDecl(name token.Token, typ T, expr T) T
	InfixDecl(assoc token.Token, prec token.Token, name token.Token) T
	This(where token.Token) T
}

type Builder struct{}

var _ Repr[Node] = Builder{}

func (b Builder) Var(name token.Token) Node {
	return &Var{Name: name}
}

func (b Builder) Literal(value token.Token) Node {
	return &Literal{Token: value}
}

func (b Builder) Paren(expr Node) Node {
	return &Paren{Expr: expr}
}

func (b Builder) Access(receiver Node, name token.Token) Node {
	return &Access{Receiver: receiver, Name: name}
}

func (b Builder) Call(callee Node, args []Node) Node {
	return &Call{Func: callee, Args: args}
}

func (b Builder) Prim(name token.Token, args []Node) Node {
	return &Prim{Name: name, Args: args}
}

func (b Builder) Binary(left Node, op token.Token, right Node) Node {
	return &Binary{Left: left, Op: op, Right: right}
}

func (b Builder) Assert(expr Node, typ Node) Node {
	return &Assert{Expr: expr, Type: typ}
}

func (b Builder) Let(bind Node, body Node) Node {
	return &Let{Bind: bind, Body: body}
}

func (b Builder) Codata(nodes []Node) Node {
	clauses := make([]*Clause, len(nodes))
	for i, node := range nodes {
		var ok bool
		clauses[i], ok = node.(*Clause)
		if !ok {
			log.Panicf("invalid node %v", node)
		}
	}

	return &Codata{Clauses: clauses}
}

func (b Builder) Clause(patterns []Node, exprs []Node) Node {
	return &Clause{Patterns: patterns, Exprs: exprs}
}

func (b Builder) Lambda(params []token.Token, exprs []Node) Node {
	return &Lambda{Params: params, Exprs: exprs}
}

func (b Builder) Case(scrutinees []Node, nodes []Node) Node {
	clauses := make([]*Clause, len(nodes))

	for i, node := range nodes {
		var ok bool
		clauses[i], ok = node.(*Clause)
		if !ok {
			log.Panicf("invalid node %v", node)
		}
	}

	return &Case{Scrutinees: scrutinees, Clauses: clauses}
}

func (b Builder) Object(nodes []Node) Node {
	fields := make([]*Field, len(nodes))

	for i, node := range nodes {
		var ok bool
		fields[i], ok = node.(*Field)
		if !ok {
			log.Panicf("invalid node %v", node)
		}
	}

	return &Object{Fields: fields}
}

func (b Builder) Field(name string, exprs []Node) Node {
	return &Field{Name: name, Exprs: exprs}
}

func (b Builder) TypeDecl(def Node, types []Node) Node {
	return &TypeDecl{Def: def, Types: types}
}

func (b Builder) VarDecl(name token.Token, typ Node, expr Node) Node {
	return &VarDecl{Name: name, Type: typ, Expr: expr}
}

func (b Builder) InfixDecl(assoc token.Token, prec token.Token, name token.Token) Node {
	return &InfixDecl{Assoc: assoc, Prec: prec, Name: name}
}

func (b Builder) This(where token.Token) Node {
	return &This{Token: where}
}
