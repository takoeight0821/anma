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

// var = IDENTIFIER ;
type Var struct {
	Name Token
}

func (v Var) String() string {
	return parenthesize("var", v.Name)
}

func (v Var) Base() Token {
	return v.Name
}

var _ Node = Var{}

// literal = INTEGER | STRING ;
type Literal struct {
	Token
}

func (l Literal) String() string {
	return parenthesize("literal", l.Token)
}

func (l Literal) Base() Token {
	return l.Token
}

var _ Node = Literal{}

// paren = "(" expr ("," expr)* ","? ")" | "(" ")" ;
type Paren struct {
	// If len(Exprs) == 0, it is an empty tuple.
	// If len(Exprs) == 1, it is a parenthesized expression.
	// Otherwise, it is a tuple.
	Elems []Node
}

func (p Paren) String() string {
	return parenthesize("paren", p.Elems...)
}

func (p Paren) Base() Token {
	if len(p.Elems) == 0 {
		return Token{}
	}
	return p.Elems[0].Base()
}

var _ Node = Paren{}

// access = call ("." IDENTIFIER)* ;
type Access struct {
	Receiver Node
	Name     Token
}

func (a Access) String() string {
	return parenthesize("access", a.Receiver, a.Name)
}

func (a Access) Base() Token {
	return a.Name
}

var _ Node = Access{}

// call = atom ("(" ")" | "(" expr ("," expr)* ","? ")")* ;
type Call struct {
	Func Node
	Args []Node
}

func (c Call) String() string {
	return parenthesize("call", prepend(c.Func, c.Args)...)
}

func (c Call) Base() Token {
	return c.Func.Base()
}

var _ Node = Call{}

// binary = access (operator access)* ;
type Binary struct {
	Left  Node
	Op    Token
	Right Node
}

func (b Binary) String() string {
	return parenthesize("binary", b.Left, b.Op, b.Right)
}

func (b Binary) Base() Token {
	return b.Op
}

var _ Node = Binary{}

// assert = binary (":" type)* ;
type Assert struct {
	Expr Node
	Type Node
}

func (a Assert) String() string {
	return parenthesize("assert", a.Expr, a.Type)
}

func (a Assert) Base() Token {
	return a.Expr.Base()
}

var _ Node = Assert{}

// let = "let" pattern "=" assert ;
type Let struct {
	Bind Node
	Body Node
}

func (l Let) String() string {
	return parenthesize("let", l.Bind, l.Body)
}

func (l Let) Base() Token {
	return l.Bind.Base()
}

var _ Node = Let{}

// codata = "{" clause ("," clause)* ","? "}" ;
type Codata struct {
	Clauses []Clause // len(Clauses) > 0
}

func (c Codata) String() string {
	return parenthesize("codata", squash(c.Clauses)...)
}

func (c Codata) Base() Token {
	if len(c.Clauses) == 0 {
		return Token{}
	}
	return c.Clauses[0].Base()
}

var _ Node = Codata{}

// clause = pattern "->" expr (";" expr)* ";"? ;
type Clause struct {
	Pattern Node
	Exprs   []Node // len(Exprs) > 0
}

func (c Clause) String() string {
	return parenthesize("clause", prepend(c.Pattern, c.Exprs)...)
}

func (c Clause) Base() Token {
	if c.Pattern == nil {
		return Token{}
	}
	return c.Pattern.Base()
}

var _ Node = Clause{}

// fn = "fn" pattern "{" expr (";" expr)* ";"? "}" ;
type Lambda struct {
	Pattern Node
	Exprs   []Node // len(Exprs) > 0
}

func (l Lambda) String() string {
	return parenthesize("lambda", prepend(l.Pattern, l.Exprs)...)
}

func (l Lambda) Base() Token {
	return l.Pattern.Base()
}

var _ Node = Lambda{}

// case = "case" expr "{" clause ("," clause)* ","? "}" ;
type Case struct {
	Scrutinee Node
	Clauses   []Clause // len(Clauses) > 0
}

func (c Case) String() string {
	return parenthesize("case", prepend(c.Scrutinee, squash(c.Clauses))...)
}

func (c Case) Base() Token {
	return c.Scrutinee.Base()
}

var _ Node = Case{}

// object = "{" field ("," field)* ","? "}" ;
type Object struct {
	Fields []Field // len(Fields) > 0
}

func (o Object) String() string {
	return parenthesize("object", squash(o.Fields)...)
}

func (o Object) Base() Token {
	return o.Fields[0].Base()
}

var _ Node = Object{}

// field = IDENTIFIER ":" expr ;
type Field struct {
	Name  string
	Exprs []Node
}

func (f Field) String() string {
	return parenthesize("field "+f.Name, f.Exprs...)
}

func (f Field) Base() Token {
	return f.Exprs[0].Base()
}

var _ Node = Field{}

// typeDecl = "type" IDENTIFIER "=" type ;
type TypeDecl struct {
	Name Token
	Type Node
}

func (t TypeDecl) String() string {
	return parenthesize("type", t.Name, t.Type)
}

func (t TypeDecl) Base() Token {
	return t.Name
}

var _ Node = TypeDecl{}

// varDecl = "def" identifier "=" expr | "def" identifier ":" type | "def" identifier ":" type "=" expr ;
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

func (v VarDecl) Base() Token {
	return v.Name
}

var _ Node = VarDecl{}

// infixDecl = ("infix" | "infixl" | "infixr") INTEGER IDENTIFIER ;
type InfixDecl struct {
	Assoc Token
	Prec  Token
	Name  Token
}

func (i InfixDecl) String() string {
	return parenthesize("infix", i.Assoc, i.Prec, i.Name)
}

func (i InfixDecl) Base() Token {
	return i.Assoc
}

var _ Node = InfixDecl{}

type This struct {
	Token
}

func (t This) String() string {
	return parenthesize("this", t.Token)
}

func (t This) Base() Token {
	return t.Token
}

var _ Node = This{}

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
// If n has children, f transforms each child before n.
// Otherwise, n is directly applied to f.
//
//tool:ignore
func Transform(n Node, f func(Node) Node) Node {
	switch n := n.(type) {
	case Var:
		return f(n)
	case Literal:
		return f(n)
	case Paren:
		for i, elem := range n.Elems {
			n.Elems[i] = Transform(elem, f)
		}
		return f(n)
	case Access:
		n.Receiver = Transform(n.Receiver, f)
		return f(n)
	case Call:
		n.Func = Transform(n.Func, f)
		for i, arg := range n.Args {
			n.Args[i] = Transform(arg, f)
		}
		return f(n)
	case Binary:
		n.Left = Transform(n.Left, f)
		n.Right = Transform(n.Right, f)
		return f(n)
	case Assert:
		n.Expr = Transform(n.Expr, f)
		n.Type = Transform(n.Type, f)
		return f(n)
	case Let:
		n.Bind = Transform(n.Bind, f)
		n.Body = Transform(n.Body, f)
		return f(n)
	case Codata:
		for i, clause := range n.Clauses {
			n.Clauses[i] = Transform(clause, f).(Clause)
		}
		return f(n)
	case Clause:
		n.Pattern = Transform(n.Pattern, f)
		for i, expr := range n.Exprs {
			n.Exprs[i] = Transform(expr, f)
		}
		return f(n)
	case Lambda:
		n.Pattern = Transform(n.Pattern, f)
		for i, expr := range n.Exprs {
			n.Exprs[i] = Transform(expr, f)
		}
		return f(n)
	case Case:
		n.Scrutinee = Transform(n.Scrutinee, f)
		for i, clause := range n.Clauses {
			n.Clauses[i] = Transform(clause, f).(Clause)
		}
		return f(n)
	case Object:
		for i, field := range n.Fields {
			n.Fields[i] = Transform(field, f).(Field)
		}
		return f(n)
	case Field:
		for i, expr := range n.Exprs {
			n.Exprs[i] = Transform(expr, f)
		}
		return f(n)
	case TypeDecl:
		n.Type = Transform(n.Type, f)
		return f(n)
	case VarDecl:
		n.Type = Transform(n.Type, f)
		n.Expr = Transform(n.Expr, f)
		return f(n)
	case InfixDecl:
		return f(n)
	case This:
		return f(n)
	default:
		return f(n)
	}
}
