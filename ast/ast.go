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
	return fmt.Sprintf("(var %s)", v.Name.Pretty())
}

func (v *Var) Base() token.Token {
	return v.Name
}

func (v *Var) Plate(f func(Node) Node) Node {
	return v
}

var _ Node = &Var{}

type Literal struct {
	token.Token
}

func (l Literal) String() string {
	return fmt.Sprintf("(literal %s)", l.Token.Pretty())
}

func (l *Literal) Base() token.Token {
	return l.Token
}

func (l *Literal) Plate(f func(Node) Node) Node {
	return l
}

var _ Node = &Literal{}

type Tuple struct {
	Elems []Node
}

func (p Tuple) String() string {
	return parenthesize("tuple", p.Elems...)
}

func (p *Tuple) Base() token.Token {
	if len(p.Elems) == 0 {
		return token.Token{}
	}
	return p.Elems[0].Base()
}

func (p *Tuple) Plate(f func(Node) Node) Node {
	for i, elem := range p.Elems {
		p.Elems[i] = f(elem)
	}
	return p
}

var _ Node = &Tuple{}

type Access struct {
	Receiver Node
	Name     token.Token
}

func (a Access) String() string {
	return fmt.Sprintf("(access %v %s)", a.Receiver, a.Name.Pretty())
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
	return parenthesize("call", prepend(c.Func, c.Args)...)
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
	args := make([]string, len(p.Args))
	for i, arg := range p.Args {
		args[i] = arg.String()
	}
	if len(args) == 0 {
		return fmt.Sprintf("(prim %s)", p.Name.Pretty())
	}
	return fmt.Sprintf("(prim %s %s)", p.Name.Pretty(), strings.Join(args, " "))
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
	return fmt.Sprintf("(binary %v %s %v)", b.Left, b.Op.Pretty(), b.Right)
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
	return parenthesize("assert", a.Expr, a.Type)
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
	return parenthesize("let", l.Bind, l.Body)
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
	Clauses []*Clause // len(Clauses) > 0
}

func (c Codata) String() string {
	return parenthesize("codata", squash(c.Clauses)...)
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
	Pattern Node
	Exprs   []Node // len(Exprs) > 0
}

func (c Clause) String() string {
	return parenthesize("clause", prepend(c.Pattern, c.Exprs)...)
}

func (c *Clause) Base() token.Token {
	if c.Pattern == nil {
		return token.Token{}
	}
	return c.Pattern.Base()
}

func (c *Clause) Plate(f func(Node) Node) Node {
	c.Pattern = f(c.Pattern)
	for i, expr := range c.Exprs {
		c.Exprs[i] = f(expr)
	}
	return c
}

var _ Node = &Clause{}

type Lambda struct {
	Pattern Node
	Exprs   []Node // len(Exprs) > 0
}

func (l Lambda) String() string {
	return parenthesize("lambda", prepend(l.Pattern, l.Exprs)...)
}

func (l *Lambda) Base() token.Token {
	return l.Pattern.Base()
}

func (l *Lambda) Plate(f func(Node) Node) Node {
	l.Pattern = f(l.Pattern)
	for i, expr := range l.Exprs {
		l.Exprs[i] = f(expr)
	}
	return l
}

var _ Node = &Lambda{}

type Case struct {
	Scrutinee Node
	Clauses   []*Clause // len(Clauses) > 0
}

func (c Case) String() string {
	return parenthesize("case", prepend(c.Scrutinee, squash(c.Clauses))...)
}

func (c *Case) Base() token.Token {
	return c.Scrutinee.Base()
}

func (c *Case) Plate(f func(Node) Node) Node {
	c.Scrutinee = f(c.Scrutinee)
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
	return parenthesize("object", squash(o.Fields)...)
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
	return parenthesize("field "+f.Name, f.Exprs...)
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
	var b strings.Builder
	b.WriteString("(type")
	b.WriteString(" ")
	b.WriteString(t.Def.String())
	for _, typ := range t.Types {
		b.WriteString(" ")
		b.WriteString(typ.String())
	}
	b.WriteString(")")
	return b.String()
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
		return fmt.Sprintf("(def %s %s)", v.Name.Pretty(), v.Expr.String())
	}
	if v.Expr == nil {
		return fmt.Sprintf("(def %s %s)", v.Name.Pretty(), v.Type.String())
	}
	return fmt.Sprintf("(def %s %s %s)", v.Name.Pretty(), v.Type.String(), v.Expr.String())
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
	return fmt.Sprintf("(infix %s %s %s)", i.Assoc.Pretty(), i.Prec.Pretty(), i.Name.Pretty())
}

func (i *InfixDecl) Base() token.Token {
	return i.Assoc
}

func (i *InfixDecl) Plate(f func(Node) Node) Node {
	return i
}

var _ Node = &InfixDecl{}

type This struct {
	token.Token
}

func (t This) String() string {
	return fmt.Sprintf("(this %s)", t.Token.Pretty())
}

func (t *This) Base() token.Token {
	return t.Token
}

func (t *This) Plate(f func(Node) Node) Node {
	return t
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
// If n is defined in ast.go and has children, Transform modifies each child before n.
// Otherwise, n is directly applied to f.
//
//tool:ignore
func Transform(n Node, f func(Node) Node) Node {
	return f(n.Plate(func(n Node) Node {
		return Transform(n, f)
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
	Transform(n, func(n Node) Node {
		nodes = append(nodes, n)
		return n
	})
	return nodes
}
