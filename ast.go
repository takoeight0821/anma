package main

import (
	"fmt"
	"strings"
)

// AST

type Node interface {
	fmt.Stringer
	Base() Token
}

type Var struct {
	Name Token
}

func (v Var) String() string {
	if v.Name.Literal == nil {
		return parenthesize("var", v.Name)
	}
	return parenthesize("var", v.Name)
}

func (v *Var) Base() Token {
	return v.Name
}

var _ Node = &Var{}

type Literal struct {
	Token
}

func (l Literal) String() string {
	return parenthesize("literal", l.Token)
}

func (l *Literal) Base() Token {
	return l.Token
}

var _ Node = &Literal{}

type Paren struct {
	// If len(Exprs) == 0, it is an empty tuple.
	// If len(Exprs) == 1, it is a parenthesized expression.
	// Otherwise, it is a tuple.
	Elems []Node
}

func (p Paren) String() string {
	return parenthesize("paren", p.Elems...)
}

func (p *Paren) Base() Token {
	if len(p.Elems) == 0 {
		return Token{}
	}
	return p.Elems[0].Base()
}

var _ Node = &Paren{}

type Access struct {
	Receiver Node
	Name     Token
}

func (a Access) String() string {
	return parenthesize("access", a.Receiver, a.Name)
}

func (a *Access) Base() Token {
	return a.Name
}

var _ Node = &Access{}

type Call struct {
	Func Node
	Args []Node
}

func (c Call) String() string {
	return parenthesize("call", prepend(c.Func, c.Args)...)
}

func (c *Call) Base() Token {
	return c.Func.Base()
}

var _ Node = &Call{}

type Binary struct {
	Left  Node
	Op    Token
	Right Node
}

func (b Binary) String() string {
	return parenthesize("binary", b.Left, b.Op, b.Right)
}

func (b *Binary) Base() Token {
	return b.Op
}

var _ Node = &Binary{}

type Assert struct {
	Expr Node
	Type Node
}

func (a Assert) String() string {
	return parenthesize("assert", a.Expr, a.Type)
}

func (a *Assert) Base() Token {
	return a.Expr.Base()
}

var _ Node = &Assert{}

type Let struct {
	Bind Node
	Body Node
}

func (l Let) String() string {
	return parenthesize("let", l.Bind, l.Body)
}

func (l *Let) Base() Token {
	return l.Bind.Base()
}

var _ Node = &Let{}

type Codata struct {
	Clauses []*Clause // len(Clauses) > 0
}

func (c Codata) String() string {
	return parenthesize("codata", squash(c.Clauses)...)
}

func (c *Codata) Base() Token {
	if len(c.Clauses) == 0 {
		return Token{}
	}
	return c.Clauses[0].Base()
}

var _ Node = &Codata{}

type Clause struct {
	Pattern Node
	Exprs   []Node // len(Exprs) > 0
}

func (c Clause) String() string {
	return parenthesize("clause", prepend(c.Pattern, c.Exprs)...)
}

func (c *Clause) Base() Token {
	if c.Pattern == nil {
		return Token{}
	}
	return c.Pattern.Base()
}

var _ Node = &Clause{}

type Lambda struct {
	Pattern Node
	Exprs   []Node // len(Exprs) > 0
}

func (l Lambda) String() string {
	return parenthesize("lambda", prepend(l.Pattern, l.Exprs)...)
}

func (l *Lambda) Base() Token {
	return l.Pattern.Base()
}

var _ Node = &Lambda{}

type Case struct {
	Scrutinee Node
	Clauses   []*Clause // len(Clauses) > 0
}

func (c Case) String() string {
	return parenthesize("case", prepend(c.Scrutinee, squash(c.Clauses))...)
}

func (c *Case) Base() Token {
	return c.Scrutinee.Base()
}

var _ Node = &Case{}

type Object struct {
	Fields []*Field // len(Fields) > 0
}

func (o Object) String() string {
	return parenthesize("object", squash(o.Fields)...)
}

func (o *Object) Base() Token {
	return o.Fields[0].Base()
}

var _ Node = &Object{}

type Field struct {
	Name  string
	Exprs []Node
}

func (f Field) String() string {
	return parenthesize("field "+f.Name, f.Exprs...)
}

func (f *Field) Base() Token {
	return f.Exprs[0].Base()
}

var _ Node = &Field{}

type TypeDecl struct {
	Name Token
	Type Node
}

func (t TypeDecl) String() string {
	return parenthesize("type", t.Name, t.Type)
}

func (t *TypeDecl) Base() Token {
	return t.Name
}

var _ Node = &TypeDecl{}

type VarDecl struct {
	Name Token
	Type Node
	Expr Node
}

func (v VarDecl) String() string {
	if v.Type == nil {
		return parenthesize("def", v.Name, v.Expr)
	}
	if v.Expr == nil {
		return parenthesize("def", v.Name, v.Type)
	}
	return parenthesize("def", v.Name, v.Type, v.Expr)
}

func (v *VarDecl) Base() Token {
	return v.Name
}

var _ Node = &VarDecl{}

type InfixDecl struct {
	Assoc Token
	Prec  Token
	Name  Token
}

func (i InfixDecl) String() string {
	return parenthesize("infix", i.Assoc, i.Prec, i.Name)
}

func (i *InfixDecl) Base() Token {
	return i.Assoc
}

var _ Node = &InfixDecl{}

type This struct {
	Token
}

func (t This) String() string {
	return parenthesize("this", t.Token)
}

func (t *This) Base() Token {
	return t.Token
}

var _ Node = &This{}

func parenthesize(head string, nodes ...Node) string {
	var b strings.Builder
	b.WriteString("(")
	b.WriteString(head)
	for _, node := range nodes {
		b.WriteString(" ")
		if node == nil {
			b.WriteString("<nil>")
		} else {
			b.WriteString(node.String())
		}
	}
	b.WriteString(")")
	return b.String()
}

func squash[T Node](elems []T) []Node {
	nodes := make([]Node, len(elems))
	for i, elem := range elems {
		nodes[i] = elem
	}
	return nodes
}

func prepend(elem Node, slice []Node) []Node {
	return append([]Node{elem}, slice...)
}

// Transform the [Node] in depth-first order.
// f is called for each node.
// If n has children, Transform modifies each child before n.
// Otherwise, n is directly applied to f.
//
//tool:ignore
func Transform(n Node, f func(Node) Node) Node {
	switch n := n.(type) {
	case *Paren:
		for i, elem := range n.Elems {
			n.Elems[i] = Transform(elem, f)
		}
	case *Access:
		n.Receiver = Transform(n.Receiver, f)
	case *Call:
		n.Func = Transform(n.Func, f)
		for i, arg := range n.Args {
			n.Args[i] = Transform(arg, f)
		}
	case *Binary:
		n.Left = Transform(n.Left, f)
		n.Right = Transform(n.Right, f)
	case *Assert:
		n.Expr = Transform(n.Expr, f)
		n.Type = Transform(n.Type, f)
	case *Let:
		n.Bind = Transform(n.Bind, f)
		n.Body = Transform(n.Body, f)
	case *Codata:
		for i, clause := range n.Clauses {
			n.Clauses[i] = Transform(clause, f).(*Clause)
		}
	case *Clause:
		n.Pattern = Transform(n.Pattern, f)
		for i, expr := range n.Exprs {
			n.Exprs[i] = Transform(expr, f)
		}
	case *Lambda:
		n.Pattern = Transform(n.Pattern, f)
		for i, expr := range n.Exprs {
			n.Exprs[i] = Transform(expr, f)
		}
	case *Case:
		n.Scrutinee = Transform(n.Scrutinee, f)
		for i, clause := range n.Clauses {
			n.Clauses[i] = Transform(clause, f).(*Clause)
		}
	case *Object:
		for i, field := range n.Fields {
			n.Fields[i] = Transform(field, f).(*Field)
		}
	case *Field:
		for i, expr := range n.Exprs {
			n.Exprs[i] = Transform(expr, f)
		}
	case *TypeDecl:
		n.Type = Transform(n.Type, f)
	case *VarDecl:
		n.Type = Transform(n.Type, f)
		n.Expr = Transform(n.Expr, f)
	}
	return f(n)
}
