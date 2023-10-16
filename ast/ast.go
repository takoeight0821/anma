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
}

// var := IDENTIFIER
type Var struct {
	Name token.Token
}

func (v Var) String() string {
	return parenthesize("var", v.Name)
}

func (v Var) Base() token.Token {
	return v.Name
}

var _ Node = Var{}

// literal := INTEGER | FLOAT | RUNE | STRING
type Literal struct {
	token.Token
}

func (l Literal) String() string {
	return parenthesize("literal", l.Token)
}

func (l Literal) Base() token.Token {
	return l.Token
}

var _ Node = Literal{}

// paren := "(" expr ("," expr)* ","? ")" | "(" ")"
// If len(Exprs) == 0, it is an empty tuple.
// If len(Exprs) == 1, it is a parenthesized expression.
// Otherwise, it is a tuple.
type Paren struct {
	Elems []Node
}

func (p Paren) String() string {
	ss := make([]fmt.Stringer, len(p.Elems))
	for i, elem := range p.Elems {
		ss[i] = elem
	}
	return parenthesize("paren", ss...)
}

func (p Paren) Base() token.Token {
	if len(p.Elems) == 0 {
		return token.Token{}
	}
	return p.Elems[0].Base()
}

var _ Node = Paren{}

// access := expr "." IDENTIFIER
type Access struct {
	Receiver Node
	Name     token.Token
}

func (a Access) String() string {
	return parenthesize("access", a.Receiver, a.Name)
}

func (a Access) Base() token.Token {
	return a.Name
}

var _ Node = Access{}

// call := expr "(" ")" | expr "(" expr ("," expr)* ","? ")"
type Call struct {
	Func Node
	Args []Node
}

func (c Call) String() string {
	return parenthesize("call", prepend(c.Func, squash(c.Args))...)
}

func (c Call) Base() token.Token {
	return c.Func.Base()
}

var _ Node = Call{}

// binary := expr operator expr
type Binary struct {
	Left  Node
	Op    token.Token
	Right Node
}

func (b Binary) String() string {
	return parenthesize("binary", b.Left, b.Op, b.Right)
}

func (b Binary) Base() token.Token {
	return b.Op
}

var _ Node = Binary{}

// assert := expr ":" type
type Assert struct {
	Expr Node
	Type Node
}

func (a Assert) String() string {
	return parenthesize("assert", a.Expr, a.Type)
}

func (a Assert) Base() token.Token {
	return a.Expr.Base()
}

var _ Node = Assert{}

// let := "let" pattern "=" expr
type Let struct {
	Bind Node
	Body Node
}

func (l Let) String() string {
	return parenthesize("let", l.Bind, l.Body)
}

func (l Let) Base() token.Token {
	return l.Bind.Base()
}

var _ Node = Let{}

// codata := "{" clause ("," clause)* ","? "}"
type Codata struct {
	Clauses []Clause // len(Clauses) > 0
}

func (c Codata) String() string {
	return parenthesize("codata", squash(c.Clauses)...)
}

func (c Codata) Base() token.Token {
	if len(c.Clauses) == 0 {
		return token.Token{}
	}
	return c.Clauses[0].Base()
}

var _ Node = Codata{}

// clause := pattern "->" expr (";" expr)* ";"?
type Clause struct {
	Pattern Node
	Exprs   []Node // len(Exprs) > 0
}

func (c Clause) String() string {
	return parenthesize("clause", prepend(c.Pattern, squash(c.Exprs))...)
}

func (c Clause) Base() token.Token {
	if c.Pattern == nil {
		return token.Token{}
	}
	return c.Pattern.Base()
}

var _ Node = Clause{}

// lambda := "fn" pattern "{" expr (";" expr)* ";"? "}"
type Lambda struct {
	Pattern Node
	Exprs   []Node // len(Exprs) > 0
}

func (l Lambda) String() string {
	return parenthesize("lambda", prepend(l.Pattern, squash(l.Exprs))...)
}

func (l Lambda) Base() token.Token {
	return l.Pattern.Base()
}

var _ Node = Lambda{}

// case := "case" expr "{" clause ("," clause)* ","? "}"
type Case struct {
	Scrutinee Node
	Clauses   []Clause // len(Clauses) > 0
}

func (c Case) String() string {
	return parenthesize("case", prepend(c.Scrutinee, squash(c.Clauses))...)
}

func (c Case) Base() token.Token {
	return c.Scrutinee.Base()
}

var _ Node = Case{}

// object := "{" field ("," field)* ","? "}"
type Object struct {
	Fields []Field // len(Fields) > 0
}

func (o Object) String() string {
	return parenthesize("object", squash(o.Fields)...)
}

func (o Object) Base() token.Token {
	return o.Fields[0].Base()
}

var _ Node = Object{}

// field := IDENTIFIER ":" expr
type Field struct {
	Name  string
	Exprs []Node
}

func (f Field) String() string {
	return parenthesize("field "+f.Name, squash(f.Exprs)...)
}

func (f Field) Base() token.Token {
	return f.Exprs[0].Base()
}

var _ Node = Field{}

// typeDecl := "type" IDENTIFIER "=" type
type TypeDecl struct {
	Name token.Token
	Type Node
}

func (t TypeDecl) String() string {
	return parenthesize("type", t.Name, t.Type)
}

func (t TypeDecl) Base() token.Token {
	return t.Name
}

var _ Node = TypeDecl{}

// varDecl := "def" identifier "=" expr | "def" identifier ":" type | "def" identifier ":" type "=" expr
type VarDecl struct {
	Name token.Token
	Type Node
	Expr Node
}

func (v VarDecl) String() string {
	if v.Type == nil {
		return parenthesize("var", v.Name, v.Expr)
	}
	if v.Expr == nil {
		return parenthesize("var", v.Name, v.Type)
	}
	return parenthesize("var", v.Name, v.Type, v.Expr)
}

func (v VarDecl) Base() token.Token {
	return v.Name
}

var _ Node = VarDecl{}

// infixDecl := ("infix" | "infixl" | "infixr") INTEGER IDENTIFIER
type InfixDecl struct {
	Assoc      token.Token
	Precedence token.Token
	Name       token.Token
}

func (i InfixDecl) String() string {
	return parenthesize("infix", i.Assoc, i.Precedence, i.Name)
}

func (i InfixDecl) Base() token.Token {
	return i.Assoc
}

var _ Node = InfixDecl{}

type This struct {
	token.Token
}

func (t This) String() string {
	return parenthesize("this", t.Token)
}

func (t This) Base() token.Token {
	return t.Token
}

var _ Node = This{}

func parenthesize(head string, nodes ...fmt.Stringer) string {
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

func squash[T fmt.Stringer](elems []T) []fmt.Stringer {
	nodes := make([]fmt.Stringer, len(elems))
	for i, elem := range elems {
		nodes[i] = elem
	}
	return nodes
}

func prepend(elem fmt.Stringer, slice []fmt.Stringer) []fmt.Stringer {
	return append([]fmt.Stringer{elem}, slice...)
}

type Kind int

const (
	KExpr   Kind = 0b000001
	KPat    Kind = 0b000010
	KType   Kind = 0b000100
	KStmt   Kind = 0b001000
	KClause Kind = 0b010000
	KField  Kind = 0b100000
	Outer   Kind = 0b000000
	Any     Kind = 0b111111
)

func (k Kind) String() string {
	var b strings.Builder
	if IsExpr(k) {
		b.WriteString("expr")
	}
	if IsPat(k) {
		if b.Len() > 0 {
			b.WriteString("|")
		}
		b.WriteString("pat")
	}
	if IsType(k) {
		if b.Len() > 0 {
			b.WriteString("|")
		}
		b.WriteString("type")
	}
	if IsStmt(k) {
		if b.Len() > 0 {
			b.WriteString("|")
		}
		b.WriteString("stmt")
	}
	if IsClause(k) {
		if b.Len() > 0 {
			b.WriteString("|")
		}
		b.WriteString("clause")
	}
	if IsField(k) {
		if b.Len() > 0 {
			b.WriteString("|")
		}
		b.WriteString("field")
	}
	if IsOuter(k) {
		if b.Len() > 0 {
			b.WriteString("|")
		}
		b.WriteString("outer")
	}
	return b.String()
}

func IsExpr(k Kind) bool {
	return k&KExpr != 0
}

func IsPat(k Kind) bool {
	return k&KPat != 0
}

func IsType(k Kind) bool {
	return k&KType != 0
}

func IsStmt(k Kind) bool {
	return k&KStmt != 0
}

func IsClause(k Kind) bool {
	return k&KClause != 0
}

func IsField(k Kind) bool {
	return k&KField != 0
}

func IsOuter(k Kind) bool {
	return k == Outer
}

// traverse the AST in depth-first order.
// f is called for each node with the node and its kind.
func Traverse(n Node, f func(Node, Kind) Node, k Kind) Node {
	switch n := n.(type) {
	case Var:
		return f(n, (KExpr|KPat|KType)&k)
	case Literal:
		return f(n, (KExpr|KPat)&k)
	case Paren:
		for i, elem := range n.Elems {
			n.Elems[i] = Traverse(elem, f, k)
		}
		return f(n, (KExpr|KPat|KType)&k)
	case Access:
		n.Receiver = Traverse(n.Receiver, f, k)
		return f(n, (KExpr|KPat)&k)
	case Call:
		n.Func = Traverse(n.Func, f, k)
		for i, arg := range n.Args {
			n.Args[i] = Traverse(arg, f, k)
		}
		return f(n, (KExpr|KPat|KType)&k)
	case Binary:
		n.Left = Traverse(n.Left, f, k)
		n.Right = Traverse(n.Right, f, k)
		return f(n, (KExpr|KType)&k)
	case Assert:
		n.Expr = Traverse(n.Expr, f, KExpr)
		n.Type = Traverse(n.Type, f, KType)
		return f(n, KExpr)
	case Let:
		n.Bind = Traverse(n.Bind, f, KPat)
		n.Body = Traverse(n.Body, f, KExpr)
		return f(n, KExpr)
	case Codata:
		for i, clause := range n.Clauses {
			n.Clauses[i] = Traverse(clause, f, KClause).(Clause)
		}
		return f(n, KExpr)
	case Clause:
		n.Pattern = Traverse(n.Pattern, f, KPat)
		for i, expr := range n.Exprs {
			n.Exprs[i] = Traverse(expr, f, KExpr)
		}
		return f(n, KClause)
	case Lambda:
		n.Pattern = Traverse(n.Pattern, f, KPat)
		for i, expr := range n.Exprs {
			n.Exprs[i] = Traverse(expr, f, KExpr)
		}
		return f(n, KExpr)
	case Case:
		n.Scrutinee = Traverse(n.Scrutinee, f, KExpr)
		for i, clause := range n.Clauses {
			n.Clauses[i] = Traverse(clause, f, KClause).(Clause)
		}
		return f(n, KExpr)
	case Object:
		for i, field := range n.Fields {
			n.Fields[i] = Traverse(field, f, KField).(Field)
		}
		return f(n, KExpr)
	case Field:
		for i, expr := range n.Exprs {
			n.Exprs[i] = Traverse(expr, f, KExpr)
		}
		return f(n, KField)
	case TypeDecl:
		n.Type = Traverse(n.Type, f, KType)
		return f(n, KStmt)
	case VarDecl:
		if n.Type != nil {
			n.Type = Traverse(n.Type, f, KType)
		}
		if n.Expr != nil {
			n.Expr = Traverse(n.Expr, f, KExpr)
		}
		return f(n, KStmt)
	case InfixDecl:
		return f(n, KStmt)
	case This:
		return f(n, KPat)
	default:
		return f(n, Outer)
	}
}
