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
	// If f returns an error, f also must return the original argument n.
	// It is similar to Visitor pattern.
	// FYI: https://hackage.haskell.org/package/lens-5.2.3/docs/Control-Lens-Plated.html
	Plate(error, func(Node, error) (Node, error)) (Node, error)
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

func (v *Var) Plate(err error, _ func(Node, error) (Node, error)) (Node, error) {
	return v, err
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

func (l *Literal) Plate(err error, _ func(Node, error) (Node, error)) (Node, error) {
	return l, err
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

func (p *Paren) Plate(err error, f func(Node, error) (Node, error)) (Node, error) {
	p.Expr, err = f(p.Expr, err)
	return p, err
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

func (a *Access) Plate(err error, f func(Node, error) (Node, error)) (Node, error) {
	a.Receiver, err = f(a.Receiver, err)
	return a, err
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

func (c *Call) Plate(err error, f func(Node, error) (Node, error)) (Node, error) {
	c.Func, err = f(c.Func, err)
	for i, arg := range c.Args {
		c.Args[i], err = f(arg, err)
	}
	return c, err
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

func (p *Prim) Plate(err error, f func(Node, error) (Node, error)) (Node, error) {
	for i, arg := range p.Args {
		p.Args[i], err = f(arg, err)
	}
	return p, err
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

func (b *Binary) Plate(err error, f func(Node, error) (Node, error)) (Node, error) {
	b.Left, err = f(b.Left, err)
	b.Right, err = f(b.Right, err)
	return b, err
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

func (a *Assert) Plate(err error, f func(Node, error) (Node, error)) (Node, error) {
	a.Expr, err = f(a.Expr, err)
	a.Type, err = f(a.Type, err)
	return a, err
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

func (l *Let) Plate(err error, f func(Node, error) (Node, error)) (Node, error) {
	l.Bind, err = f(l.Bind, err)
	l.Body, err = f(l.Body, err)
	return l, err
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

func (c *Codata) Plate(err error, f func(Node, error) (Node, error)) (Node, error) {
	for i, clause := range c.Clauses {
		var cl Node
		cl, err = f(clause, err)
		c.Clauses[i] = cl.(*Clause)
	}
	return c, err
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

func (c *Clause) Plate(err error, f func(Node, error) (Node, error)) (Node, error) {
	for i, pattern := range c.Patterns {
		c.Patterns[i], err = f(pattern, err)
	}
	for i, expr := range c.Exprs {
		c.Exprs[i], err = f(expr, err)
	}
	return c, err
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

func (l *Lambda) Plate(err error, f func(Node, error) (Node, error)) (Node, error) {
	for i, expr := range l.Exprs {
		l.Exprs[i], err = f(expr, err)
	}
	return l, err
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

func (c *Case) Plate(err error, f func(Node, error) (Node, error)) (Node, error) {
	for i, scrutinee := range c.Scrutinees {
		c.Scrutinees[i], err = f(scrutinee, err)
	}
	for i, clause := range c.Clauses {
		var cl Node
		cl, err = f(clause, err)
		c.Clauses[i] = cl.(*Clause)
	}
	return c, err
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

func (o *Object) Plate(err error, f func(Node, error) (Node, error)) (Node, error) {
	for i, field := range o.Fields {
		var fl Node
		fl, err = f(field, err)
		o.Fields[i] = fl.(*Field)
	}
	return o, err
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

func (f *Field) Plate(err error, g func(Node, error) (Node, error)) (Node, error) {
	for i, expr := range f.Exprs {
		f.Exprs[i], err = g(expr, err)
	}
	return f, err
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

func (t *TypeDecl) Plate(err error, f func(Node, error) (Node, error)) (Node, error) {
	t.Def, err = f(t.Def, err)
	for i, typ := range t.Types {
		t.Types[i], err = f(typ, err)
	}
	return t, err
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

func (v *VarDecl) Plate(err error, f func(Node, error) (Node, error)) (Node, error) {
	if v.Type != nil {
		v.Type, err = f(v.Type, err)
	}
	if v.Expr != nil {
		v.Expr, err = f(v.Expr, err)
	}
	return v, err
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

func (i *InfixDecl) Plate(err error, _ func(Node, error) (Node, error)) (Node, error) {
	return i, err
}

var _ Node = &InfixDecl{}

type This struct {
	token.Token
}

func (t This) String() string {
	return "#"
}

func (t *This) Base() token.Token {
	return t.Token
}

func (t *This) Plate(err error, _ func(Node, error) (Node, error)) (Node, error) {
	return t, err
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
// If f returns an error, f also must return the original argument n.
// If n is defined in ast.go and has children, Traverse modifies each child before n.
// Otherwise, n is directly applied to f.
//
//tool:ignore
func Traverse(n Node, f func(Node, error) (Node, error)) (Node, error) {
	n, err := n.Plate(nil, func(n Node, err error) (Node, error) {
		return Traverse(n, f)
	})
	return f(n, err)
}

//tool:ignore
func Children(n Node) []Node {
	var children []Node
	_, err := n.Plate(nil, func(n Node, _ error) (Node, error) {
		children = append(children, n)
		return n, nil
	})
	if err != nil {
		panic(fmt.Errorf("unexpected error: %w", err))
	}
	return children
}

//tool:ignore
func Universe(n Node) []Node {
	var nodes []Node
	_, err := Traverse(n, func(n Node, _ error) (Node, error) {
		nodes = append(nodes, n)
		return n, nil
	})
	if err != nil {
		panic(fmt.Errorf("unexpected error: %w", err))
	}
	return nodes
}
