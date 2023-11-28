package ast

import (
	"fmt"
	"strings"

	"github.com/takoeight0821/anma/token"
)

// AST

type Node interface {
	fmt.Stringer
	Base() token.Token
	// Plate applies the given function to each child node.
	// It is similar to Visitor pattern.
	// FYI: https://hackage.haskell.org/package/lens-5.2.3/docs/Control-Lens-Plated.html
	Plate(func(Node) Node) Node
}

type Var struct {
	Name token.Token
}

func (v Var) String() string {
	return parenthesize("var", v.Name).String()
}

func (v *Var) Base() token.Token {
	return v.Name
}

func (v *Var) Plate(_ func(Node) Node) Node {
	return v
}

var _ Node = &Var{}

type Literal struct {
	token.Token
}

func (l Literal) String() string {
	return parenthesize("literal", l.Token).String()
}

func (l *Literal) Base() token.Token {
	return l.Token
}

func (l *Literal) Plate(_ func(Node) Node) Node {
	return l
}

var _ Node = &Literal{}

type Paren struct {
	Expr Node
}

func (p Paren) String() string {
	return parenthesize("paren", p.Expr).String()
}

func (p *Paren) Base() token.Token {
	return p.Expr.Base()
}

func (p *Paren) Plate(f func(Node) Node) Node {
	p.Expr = f(p.Expr)
	return p
}

var _ Node = &Paren{}

type Access struct {
	Receiver Node
	Name     token.Token
}

func (a Access) String() string {
	return parenthesize("access", a.Receiver, a.Name).String()
}

func (a *Access) Base() token.Token {
	return a.Name
}

func (a *Access) Plate(f func(Node) Node) Node {
	a.Receiver = f(a.Receiver)
	return a
}

var _ Node = &Access{}

type Call struct {
	Func Node
	Args []Node
}

func (c Call) String() string {
	return parenthesize("call", c.Func, concat(c.Args)).String()
}

func (c *Call) Base() token.Token {
	return c.Func.Base()
}

func (c *Call) Plate(f func(Node) Node) Node {
	c.Func = f(c.Func)
	for i, arg := range c.Args {
		c.Args[i] = f(arg)
	}
	return c
}

var _ Node = &Call{}

type Prim struct {
	Name token.Token
	Args []Node
}

func (p Prim) String() string {
	return parenthesize("prim", p.Name, concat(p.Args)).String()
}

func (p *Prim) Base() token.Token {
	return p.Name
}

func (p *Prim) Plate(f func(Node) Node) Node {
	for i, arg := range p.Args {
		p.Args[i] = f(arg)
	}
	return p
}

var _ Node = &Prim{}

type Binary struct {
	Left  Node
	Op    token.Token
	Right Node
}

func (b Binary) String() string {
	return parenthesize("binary", b.Left, b.Op, b.Right).String()
}

func (b *Binary) Base() token.Token {
	return b.Op
}

func (b *Binary) Plate(f func(Node) Node) Node {
	b.Left = f(b.Left)
	b.Right = f(b.Right)
	return b
}

var _ Node = &Binary{}

type Assert struct {
	Expr Node
	Type Node
}

func (a Assert) String() string {
	return parenthesize("assert", a.Expr, a.Type).String()
}

func (a *Assert) Base() token.Token {
	return a.Expr.Base()
}

func (a *Assert) Plate(f func(Node) Node) Node {
	a.Expr = f(a.Expr)
	a.Type = f(a.Type)
	return a
}

var _ Node = &Assert{}

type Let struct {
	Bind Node
	Body Node
}

func (l Let) String() string {
	return parenthesize("let", l.Bind, l.Body).String()
}

func (l *Let) Base() token.Token {
	return l.Bind.Base()
}

func (l *Let) Plate(f func(Node) Node) Node {
	l.Bind = f(l.Bind)
	l.Body = f(l.Body)
	return l
}

var _ Node = &Let{}

type Codata struct {
	// len(Clauses) > 0
	// for each clause, len(Patterns) == 1
	Clauses []*Clause
}

func (c Codata) String() string {
	return parenthesize("codata", concat(c.Clauses)).String()
}

func (c *Codata) Base() token.Token {
	if len(c.Clauses) == 0 {
		return token.Token{}
	}
	return c.Clauses[0].Base()
}

func (c *Codata) Plate(f func(Node) Node) Node {
	for i, clause := range c.Clauses {
		c.Clauses[i] = f(clause).(*Clause)
	}
	return c
}

var _ Node = &Codata{}

type Clause struct {
	Patterns []Node
	Exprs    []Node // len(Exprs) > 0
}

func (c Clause) String() string {
	var pat fmt.Stringer
	if len(c.Patterns) > 1 {
		pat = parenthesize("", concat(c.Patterns))
	} else {
		pat = c.Patterns[0]
	}

	return parenthesize("clause", pat, concat(c.Exprs)).String()
}

func (c *Clause) Base() token.Token {
	if len(c.Patterns) > 0 {
		return c.Patterns[0].Base()
	}
	return c.Exprs[0].Base()
}

func (c *Clause) Plate(f func(Node) Node) Node {
	for i, pattern := range c.Patterns {
		c.Patterns[i] = f(pattern)
	}
	for i, expr := range c.Exprs {
		c.Exprs[i] = f(expr)
	}
	return c
}

var _ Node = &Clause{}

type Lambda struct {
	Params []token.Token
	Exprs  []Node // len(Exprs) > 0
}

func (l Lambda) String() string {
	return parenthesize("lambda", parenthesize("", concat(l.Params)), concat(l.Exprs)).String()
}

func (l *Lambda) Base() token.Token {
	return l.Params[0]
}

func (l *Lambda) Plate(f func(Node) Node) Node {
	for i, expr := range l.Exprs {
		l.Exprs[i] = f(expr)
	}
	return l
}

var _ Node = &Lambda{}

type Case struct {
	Scrutinees []Node
	Clauses    []*Clause // len(Clauses) > 0
}

func (c Case) String() string {
	return parenthesize("case", parenthesize("", concat(c.Scrutinees)), concat(c.Clauses)).String()
}

func (c *Case) Base() token.Token {
	return c.Scrutinees[0].Base()
}

func (c *Case) Plate(f func(Node) Node) Node {
	for i, scrutinee := range c.Scrutinees {
		c.Scrutinees[i] = f(scrutinee)
	}
	for i, clause := range c.Clauses {
		c.Clauses[i] = f(clause).(*Clause)
	}
	return c
}

var _ Node = &Case{}

type Object struct {
	Fields []*Field // len(Fields) > 0
}

func (o Object) String() string {
	return parenthesize("object", concat(o.Fields)).String()
}

func (o *Object) Base() token.Token {
	return o.Fields[0].Base()
}

func (o *Object) Plate(f func(Node) Node) Node {
	for i, field := range o.Fields {
		o.Fields[i] = f(field).(*Field)
	}
	return o
}

var _ Node = &Object{}

type Field struct {
	Name  string
	Exprs []Node
}

func (f Field) String() string {
	return parenthesize("field "+f.Name, concat(f.Exprs)).String()
}

func (f *Field) Base() token.Token {
	return f.Exprs[0].Base()
}

func (f *Field) Plate(g func(Node) Node) Node {
	for i, expr := range f.Exprs {
		f.Exprs[i] = g(expr)
	}
	return f
}

var _ Node = &Field{}

type TypeDecl struct {
	Def   Node
	Types []Node
}

func (t TypeDecl) String() string {
	return parenthesize("type", t.Def, concat(t.Types)).String()
}

func (t *TypeDecl) Base() token.Token {
	return t.Def.Base()
}

func (t *TypeDecl) Plate(f func(Node) Node) Node {
	t.Def = f(t.Def)
	for i, typ := range t.Types {
		t.Types[i] = f(typ)
	}
	return t
}

var _ Node = &TypeDecl{}

type VarDecl struct {
	Name token.Token
	Type Node
	Expr Node
}

func (v VarDecl) String() string {
	if v.Type == nil {
		return parenthesize("def", v.Name, v.Expr).String()
	}
	if v.Expr == nil {
		return parenthesize("def", v.Name, v.Type).String()
	}
	return parenthesize("def", v.Name, v.Type, v.Expr).String()
}

func (v *VarDecl) Base() token.Token {
	return v.Name
}

func (v *VarDecl) Plate(f func(Node) Node) Node {
	if v.Type != nil {
		v.Type = f(v.Type)
	}
	if v.Expr != nil {
		v.Expr = f(v.Expr)
	}
	return v
}

var _ Node = &VarDecl{}

type InfixDecl struct {
	Assoc token.Token
	Prec  token.Token
	Name  token.Token
}

func (i InfixDecl) String() string {
	return parenthesize("infix", i.Assoc, i.Prec, i.Name).String()
}

func (i *InfixDecl) Base() token.Token {
	return i.Assoc
}

func (i *InfixDecl) Plate(_ func(Node) Node) Node {
	return i
}

var _ Node = &InfixDecl{}

type This struct {
	token.Token
}

func (t This) String() string {
	return parenthesize("this", t.Token).String()
}

func (t *This) Base() token.Token {
	return t.Token
}

func (t *This) Plate(_ func(Node) Node) Node {
	return t
}

var _ Node = &This{}

// parenthesize takes a head string and a variadic number of nodes that implement the fmt.Stringer interface.
// It returns a fmt.Stringer that represents a string where each node is parenthesized and separated by a space.
// If the head string is not empty, it is added at the beginning of the string.
//
//tool:ignore
func parenthesize(head string, elems ...fmt.Stringer) fmt.Stringer {
	var b strings.Builder
	b.WriteString("(")
	elemsStr := concat(elems).String()
	if head != "" {
		b.WriteString(head)
	}
	if elemsStr != "" {
		if head != "" {
			b.WriteString(" ")
		}
		b.WriteString(elemsStr)
	}
	b.WriteString(")")
	return &b
}

// concat takes a slice of nodes that implement the fmt.Stringer interface.
// It returns a fmt.Stringer that represents a string where each node is separated by a space.
//
//tool:ignore
func concat[T fmt.Stringer](elems []T) fmt.Stringer {
	var b strings.Builder
	for i, elem := range elems {
		// ignore empty string
		// e.g. concat({}) == ""
		str := elem.String()
		if str == "" {
			continue
		}
		if i != 0 {
			b.WriteString(" ")
		}
		b.WriteString(str)
	}
	return &b
}

// Traverse the [Node] in depth-first order.
// f is called for each node.
// If n is defined in ast.go and has children, Traverse modifies each child before n.
// Otherwise, n is directly applied to f.
//
//tool:ignore
func Traverse(n Node, f func(Node) Node) Node {
	return f(n.Plate(func(n Node) Node {
		return Traverse(n, f)
	}))
}

//tool:ignore
func Children(n Node) []Node {
	var children []Node
	n.Plate(func(n Node) Node {
		children = append(children, n)
		return n
	})
	return children
}

//tool:ignore
func Universe(n Node) []Node {
	var nodes []Node
	Traverse(n, func(n Node) Node {
		nodes = append(nodes, n)
		return n
	})
	return nodes
}
